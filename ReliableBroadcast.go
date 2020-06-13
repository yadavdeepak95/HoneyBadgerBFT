package honeybadgerbft

import (
	"crypto/sha256"
	"math"

	ab "github.com/yadavdeepak95/HoneyBadgerBFT/proto/orderer"

	"github.com/klauspost/reedsolomon"
)

type ReliableBroadcast struct {
	instanceIndex int // == leaderIndex
	total         int
	tolerance     int
	ordererIndex  int
	leaderIndex   int
	channel       chan *ab.HoneyBadgerBFTMessage
	sendFunc      func(index int, msg *ab.HoneyBadgerBFTMessageReliableBroadcast)
	broadcastFunc func(msg *ab.HoneyBadgerBFTMessageReliableBroadcast)
	enc           reedsolomon.Encoder
	In            chan []byte
	Out           chan []byte
	exitRecv      chan interface{}
}

func NewReliableBroadcast(instanceIndex int, total int, tolerance int, ordererIndex int, leaderIndex int, receiveMessageChannel chan *ab.HoneyBadgerBFTMessage, sendFunc func(index int, msg ab.HoneyBadgerBFTMessage), broadcastFunc func(msg ab.HoneyBadgerBFTMessage)) (result *ReliableBroadcast) {
	// TODO: check param relations
	s := func(index int, msg *ab.HoneyBadgerBFTMessageReliableBroadcast) {
		sendFunc(index, ab.HoneyBadgerBFTMessage{ReliableBroadcast: msg})
	}
	bc := func(msg *ab.HoneyBadgerBFTMessageReliableBroadcast) {
		broadcastFunc(ab.HoneyBadgerBFTMessage{ReliableBroadcast: msg})
	}
	var K = total - 2*tolerance
	enc, err := reedsolomon.New(K, total-K)
	if err != nil {
		logger.Panicf("Encoder failed: err = %v", err)
	}
	result = &ReliableBroadcast{
		instanceIndex: instanceIndex,
		total:         total,
		tolerance:     tolerance,
		ordererIndex:  ordererIndex,
		leaderIndex:   leaderIndex,
		channel:       receiveMessageChannel,
		sendFunc:      s,
		broadcastFunc: bc,
		enc:           enc,
		In:            make(chan []byte),
		Out:           make(chan []byte),
		exitRecv:      make(chan interface{}),
	}
	go result.reliableBroadcastService()
	logger.Debugf("RBC orderer[%v] instactance[%v] Leader[%v]", ordererIndex, instanceIndex, leaderIndex)
	return result
}

func (rbc *ReliableBroadcast) reliableBroadcastService() {
	// K               = N - 2 * f  # Need this many to reconstruct
	// EchoThreshold   = N - f      # Wait for this many ECHO to send READY
	// ReadyThreshold  = f + 1      # Wait for this many READY to amplify READY
	// OutputThreshold = 2 * f + 1  # Wait for this many READY to output
	// # NOTE: The above thresholds  are chosen to minimize the size
	// # of the erasure coding stripes, i.e. to maximize K.
	// # The following alternative thresholds are more canonical
	// # (e.g., in Bracha '86) and require larger stripes, but must wait
	// # for fewer nodes to respond
	// #   EchoThreshold = ceil((N + f + 1.)/2)
	// #   K = EchoThreshold - f
	var K = rbc.total - 2*rbc.tolerance
	var EchoThreshold = rbc.total - rbc.tolerance
	var ReadyThreshold = rbc.tolerance + 1
	var OutputThreshold = 2*rbc.tolerance + 1

	if rbc.leaderIndex == rbc.ordererIndex {
		data := <-rbc.In
		logger.Debugf("RBC[%v] input: []bytes(len=%v)", rbc.instanceIndex, len(data))
		blocks, err := Encode(rbc.enc, data)
		if err != nil {
			logger.Panicf("Error occured when encoding data: %s", err)
		}
		tree := newMerkleTree(blocks) //TODO: check whether full binary tree
		rootHash := tree[1]

		for i := 0; i < rbc.total; i++ {
			branch := getMerkleTreeBranch(tree, i)
			rbc.sendFunc(i, &ab.HoneyBadgerBFTMessageReliableBroadcast{Val: &ab.HoneyBadgerBFTMessageReliableBroadcastVAL{}, PadLength: uint64(len(data)), Block: blocks[i], RootHash: rootHash, Branch: branch})

		}
		//TODO : for testing only Comment it later
		// rbc.Out <- data
	}
	// TODO: filter policy: if leader, discard all messages until sending VAL

	var rootHashFromLeader []byte
	var blocks = make(map[string][][]byte)
	var echoCounter = make(map[string]int)
	var echoSenders = make(map[int]bool)
	var ready = make(map[string]map[int]bool)
	var readySent bool
	var readySenders = make(map[int]bool)

	decodeAndVerifyAndOutput := func(rootHash []byte, padlen int) {
		err := rbc.enc.Reconstruct(blocks[string(rootHash)])

		if err != nil {
			logger.Panicf("Error occured when decoding data: , err")
		}
		var value []byte
		for _, data := range (blocks[string(rootHash)])[:K] {
			value = append(value, data...)
		}

		logger.Debugf("RBC[%v] output: []bytes(len=%v)", rbc.instanceIndex, len(value))
		rbc.Out <- value[:padlen]
	}
	for {
		select {
		case <-rbc.exitRecv:
			return
		case msg := <-rbc.channel:

			sender := int(msg.GetSender())
			subMsg := msg.GetReliableBroadcast()
			rootHash := subMsg.GetRootHash()
			rootHashString := string(rootHash)
			branch := subMsg.GetBranch()
			block := subMsg.GetBlock()
			padlen := subMsg.GetPadLength()
			// logger.Debugf("Sender[%v] -- Recvier[%v]", sender, rbc.ordererIndex)
			// switch subMsg.Type.(type) {
			// case *ab.HoneyBadgerBFTMessageReliableBroadcast_Val:
			if subMsg.GetVal() != nil {
				// logger.Debugf("Propose recived%v", msg)
				if rootHashFromLeader != nil {
					continue
				}

				if sender != rbc.leaderIndex {
					logger.Panicf("VAL message from other than leader: %v", sender)
					continue
				}

				if !verifyMerkleTree(rootHash, branch, block, rbc.ordererIndex) {
					logger.Panicf("Failed to validate VAL message")
				}

				rootHashFromLeader = rootHash

				rbc.broadcastFunc(&ab.HoneyBadgerBFTMessageReliableBroadcast{Echo: &ab.HoneyBadgerBFTMessageReliableBroadcastECHO{}, PadLength: padlen, Block: block, RootHash: rootHash, Branch: branch})
			} else if subMsg.GetEcho() != nil {
				// case *ab.HoneyBadgerBFTMessageReliableBroadcast_Echo:
				if _, exist := blocks[rootHashString]; exist {
					if blocks[rootHashString][sender] != nil || echoSenders[sender] {
						logger.Debugf("Redundant ECHO")
						continue
					}
				} else {
					blocks[rootHashString] = make([][]byte, rbc.total)
				}

				if !verifyMerkleTree(rootHash, branch, block, sender) {
					logger.Panicf("Failed to validate ECHO message")
				}

				blocks[rootHashString][sender] = block
				echoSenders[sender] = true
				echoCounter[rootHashString]++

				//logger.Infof("RBC INST %v received a ECHO message from %v; echoCounter[roothash]=%v of %v", rbc.leaderIndex, sender, echoCounter[rootHashString], EchoThreshold)
				if echoCounter[rootHashString] >= EchoThreshold && !readySent {
					rbc.broadcastFunc(&ab.HoneyBadgerBFTMessageReliableBroadcast{Ready: &ab.HoneyBadgerBFTMessageReliableBroadcastREADY{}, RootHash: rootHash, PadLength: padlen})
					readySent = true
				}

				if len(ready[rootHashString]) >= OutputThreshold && echoCounter[rootHashString] >= K {
					decodeAndVerifyAndOutput(rootHash, int(padlen))
					return
				}
			} else if subMsg.GetReady() != nil {
				// case *ab.HoneyBadgerBFTMessageReliableBroadcast_Ready:
				_, exist := ready[rootHashString]
				if (exist && ready[rootHashString][sender]) || readySenders[sender] {
					logger.Debugf("Redundant READY")
					continue
				}

				if !exist {
					ready[rootHashString] = make(map[int]bool)
				}
				ready[rootHashString][sender] = true
				readySenders[sender] = true

				//logger.Infof("RBC INST %v received a READY message from %v; len(ready[rootHashString])=%v of %v", rbc.leaderIndex, sender, len(ready[rootHashString]), ReadyThreshold)
				if len(ready[rootHashString]) >= ReadyThreshold && !readySent {
					rbc.broadcastFunc(&ab.HoneyBadgerBFTMessageReliableBroadcast{Ready: &ab.HoneyBadgerBFTMessageReliableBroadcastREADY{}, RootHash: rootHash, PadLength: padlen})
					readySent = true
				}

				if len(ready[rootHashString]) >= OutputThreshold && echoCounter[rootHashString] >= K {
					decodeAndVerifyAndOutput(rootHash, int(padlen))
					return
				}
			}

		}
	}
}

//////////////////////////////////////////////////////////////////////////////////
//                                                                              //
//                                  TOOLS                                       //
//                                                                              //
//////////////////////////////////////////////////////////////////////////////////

func Encode(enc reedsolomon.Encoder, data []byte) ([][]byte, error) {
	shards, err := enc.Split(data)
	if err != nil {
		logger.Panic(err)
	}
	if err := enc.Encode(shards); err != nil {
		logger.Panic(err)
	}
	return shards, nil
}

func merkleTreeHash(data []byte, others ...[]byte) []byte { // NOTE: root at index=1
	s := sha256.New()
	s.Write(data)
	for _, d := range others {
		s.Write(d)
	}
	return s.Sum(nil)
}

func newMerkleTree(blocks [][]byte) [][]byte {
	bottomRow := int(math.Pow(2, math.Ceil(math.Log2(float64(len(blocks))))))
	result := make([][]byte, 2*bottomRow, 2*bottomRow)
	for i := 0; i < len(blocks); i++ {
		result[bottomRow+i] = merkleTreeHash(blocks[i])
	}
	for i := bottomRow - 1; i > 0; i-- {
		result[i] = merkleTreeHash(result[i*2], result[i*2+1])
	}
	return result
}

func getMerkleTreeBranch(tree [][]byte, index int) (result [][]byte) { // NOTE: index from 0, block index not tree item index
	t := index + (len(tree) >> 1)
	for t > 1 {
		result = append(result, tree[t^1])
		t /= 2
	}
	return result
}

func verifyMerkleTree(rootHash []byte, branch [][]byte, block []byte, index int) bool {
	//TODO: add checks
	tmp := merkleTreeHash(block)
	for _, node := range branch {
		if index&1 == 0 {
			tmp = merkleTreeHash(tmp, node)
		} else {
			tmp = merkleTreeHash(node, tmp)
		}
		index /= 2
	}
	return string(rootHash) == string(tmp)
}
