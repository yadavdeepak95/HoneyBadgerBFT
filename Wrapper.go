package honeybadgerbft

import (
	"math/rand"
	"sync"
	"time"

	cb "github.com/hyperledger/fabric/protos/common"
	ab "github.com/yadavdeepak95/HoneyBadgerBFT/proto/orderer"

	"github.com/op/go-logging"
)

const pkgLogID = "orderer/HoneybadgerBFT"

var logger = logging.MustGetLogger(pkgLogID)

type ChainImpl struct {
	Total         int
	Tolerance     int
	Index         int
	height        uint64
	batchSize     uint64
	txbuffer      []*cb.Envelope
	heightChan    map[uint64](chan *ab.HoneyBadgerBFTMessage)
	newBlockChan  chan interface{}
	bufferLock    sync.Mutex
	replayChan    chan interface{}
	heightMapLock sync.Mutex
	heightLock    sync.Mutex
	blockOut      chan interface{}
	msgChan       *MessageChannels
}

//used to maintain to call for next proposal
var tempCount uint64

//Enqueue txn input and trigger for new block
func (ch *ChainImpl) Enqueue(env *cb.Envelope) {
	ch.bufferLock.Lock()
	tempCount++
	ch.txbuffer = append(ch.txbuffer, env)
	if tempCount == ch.batchSize {
		tempCount = 0
		ch.newBlockChan <- nil
	}
	ch.bufferLock.Unlock()
}

//create new insatance of chain
func NewWrapper(batchSize uint64, ConnectionList []string, Index int, total int, tolerance int, random bool, fixed bool, fixedTime int) *ChainImpl {
	//Net Package opening port and returns channels
	//TODO : add chain id here if req
	logging.SetLevel(4, pkgLogID)
	logger.Infof("inFO LEVEL SET fixed[%v]", fixed)
	messageChannels, err := Register("1", ConnectionList, Index, random, fixed, fixedTime)
	if err != nil {
		logger.Panicf("Can not start chain: %s", err)
	}

	ch := &ChainImpl{
		batchSize:    batchSize,
		Total:        total,
		Tolerance:    tolerance,
		Index:        Index,
		msgChan:      &messageChannels,
		height:       0,
		newBlockChan: make(chan interface{}, 666666),
		blockOut:     make(chan interface{}),
		replayChan:   make(chan interface{}, 666666),
		heightChan:   make(map[uint64]chan *ab.HoneyBadgerBFTMessage),
	}
	//Send msg to channel according to height
	go ch.run()
	//main function does all main stuff propose/consensus and all
	go ch.Consensus()
	// mapping message to proper channel
	dispatchMessageByHeightService := func() {
		for {
			msg := <-ch.replayChan
			ch.heightLock.Lock()
			if msg.(*ab.HoneyBadgerBFTMessage).GetHeight() >= ch.height {
				ch.getByHeight(msg.(*ab.HoneyBadgerBFTMessage).GetHeight()) <- msg.(*ab.HoneyBadgerBFTMessage)
			}
			ch.heightLock.Unlock()
		}
	}
	go dispatchMessageByHeightService()

	return ch

}

//maintain map of channels according to height
func (ch *ChainImpl) getByHeight(height uint64) chan *ab.HoneyBadgerBFTMessage {
	ch.heightMapLock.Lock()
	defer ch.heightMapLock.Unlock()
	result, exist := ch.heightChan[height]
	if !exist {
		//TODO: calculate height according to number of nodes
		result = make(chan *ab.HoneyBadgerBFTMessage, 6666)
		ch.heightChan[height] = result
	}

	return result
}
func (ch *ChainImpl) run() {
	//if hb message send to HB it will handle according to height
	//DONE if tx in check for no of txn overbatch with counter if count excced call block cutting --DONE
	//DONE if required block out can also trigger new block (if there is no block in and no of txn are more then block limit it should trigger)
	for {
		msg := <-ch.msgChan.Receive
		ch.replayChan <- msg
	}

}

func (ch *ChainImpl) Consensus() {
	for {

		sendFunc := func(index int, msg ab.HoneyBadgerBFTMessage) {
			msg.Height = ch.height
			msg.Receiver = uint64(index)
			ch.msgChan.Send <- msg
		}
		broadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
			for i := 0; i < ch.Total; i++ {
				sendFunc(i, msg)
			}
		}
		select {
		// case <-ch.exitChan:
		// 	logger.Debug("Exiting")
		// 	return
		case <-ch.newBlockChan:
			var exitHeight = make(chan interface{})
			ch.heightLock.Lock()
			logger.Debugf("\n\n***************%v*****************\n\n", ch.height)
			ch.heightLock.Unlock()
			startTime := time.Now()
			var abaRecvMsgChannel = make(chan *ab.HoneyBadgerBFTMessage)
			var rbcRecvMsgChannel = make(chan *ab.HoneyBadgerBFTMessage)

			///////////////////////////////
			//Distribute msg by serivce
			//////////////////////////////
			dispatchByTypeService := func() {
				for {
					select {
					case <-exitHeight:
						return
					case msg := <-ch.getByHeight(ch.height):

						if msg.GetBinaryAgreement() != nil {
							abaRecvMsgChannel <- msg
						} else if msg.GetReliableBroadcast() != nil {
							rbcRecvMsgChannel <- msg
						} else {
							logger.Debugf("MSG in Serive")
						}

					}
				}
			}
			go dispatchByTypeService()

			//////////////////////////////
			//channel for every instance
			//////////////////////////////
			var abaInstanceRecvMsgChannels = make([]chan *ab.HoneyBadgerBFTMessage, ch.Total)
			var rbcInstanceRecvMsgChannels = make([]chan *ab.HoneyBadgerBFTMessage, ch.Total)

			for i := 0; i < ch.Total; i++ {
				abaInstanceRecvMsgChannels[i] = make(chan *ab.HoneyBadgerBFTMessage, 666666)
				rbcInstanceRecvMsgChannels[i] = make(chan *ab.HoneyBadgerBFTMessage, 666666)
			}

			dispatchByInstance := func() {
				for {
					select {
					case <-exitHeight:
						return
					case msg := <-abaRecvMsgChannel:
						abaInstanceRecvMsgChannels[msg.GetInstance()] <- msg
					case msg := <-rbcRecvMsgChannel:
						rbcInstanceRecvMsgChannels[msg.GetInstance()] <- msg
					}
				}
			}
			go dispatchByInstance()

			//////////////////////////
			//setting up RBC ABA
			//////////////////////////
			var aba = make([]*BinaryAgreement, ch.Total)
			var rbc = make([]*ReliableBroadcast, ch.Total)
			for i := 0; i < ch.Total; i++ {
				instanceIndex := i // NOTE important to copy i
				componentSendFunc := func(index int, msg ab.HoneyBadgerBFTMessage) {

					msg.Instance = uint64(instanceIndex)
					logger.Debugf("msg.Instance[%v],orderer[%v]", msg.Instance, ch.Index, instanceIndex)
					sendFunc(index, msg)
				}
				componentBroadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
					msg.Instance = uint64(instanceIndex)
					broadcastFunc(msg)
				}
				rbc[i] = NewReliableBroadcast(i, ch.Total, ch.Tolerance, ch.Index, i, rbcInstanceRecvMsgChannels[i], componentSendFunc, componentBroadcastFunc) // TODO: a better way to eval whether i is leader, ListenAddress may difference with address in connectionlist
				aba[i] = NewBinaryAgreement(i, ch.Total, ch.Tolerance, abaInstanceRecvMsgChannels[i], componentBroadcastFunc)                                   // May stop automatically?				                                                                                                                                                                    //TODO
			}

			///////////////////////////////////////
			//setup ACS component (using N instances of COIN ABA RBC)
			///////////////////////////////////////
			acs := NewCommonSubset(ch.Index, ch.Total, ch.Tolerance, rbc, aba)

			////////////////////////////////////////////////
			//setup HoneyBadgerBFT component (using ACS)
			////////////////////////////////////////////////
			block := NewHoneyBadgerBlock(ch.Total, ch.Tolerance, acs)
			//TODO : Improve ??
			randomSelectFunc := func(batch []*cb.Envelope, number int) (result []*cb.Envelope) {
				result = batch[:]
				if len(batch) <= number {
					return result
				}
				for len(result) > number {
					i := rand.Intn(len(batch))
					result = append(result[:i], result[i+1:]...)
				}
				return result
			}

			ch.bufferLock.Lock()
			proposalBatch := ch.txbuffer[:]
			ch.bufferLock.Unlock()
			if uint32(len(proposalBatch)) > uint32(ch.batchSize) {
				ch.bufferLock.Lock()
				proposalBatch = ch.txbuffer[:ch.batchSize]
				ch.bufferLock.Unlock()

			}
			proposalBatch = randomSelectFunc(proposalBatch, int(ch.batchSize))

			////////////////////////////////////////////
			//generate blocks
			///////////////////////////////////////////

			var committedBatch []*cb.Envelope

			block.In <- proposalBatch
			committedBatch = <-block.Out

			// <-ch.blockOut

			//Clear buffer
			if len(committedBatch) == 0 {
				logger.Warningf("No transaction committed!")
			} else {
				ch.bufferLock.Lock()
				for _, tx := range committedBatch {
					logger.Debugf("%v", tx)

					for i, v := range ch.txbuffer {
						if v.String() == tx.String() {
							//delete tx
							ch.txbuffer = append(ch.txbuffer[:i], ch.txbuffer[i+1:]...)
						}
					}
				}
				ch.bufferLock.Unlock()
			}

			ch.heightLock.Lock()
			logger.Infof("%v, %v, %v", ch.height, len(committedBatch), time.Since(startTime))

			logger.Debugf("\n\n********************Bufferlen == %v**************\n\n", len(ch.txbuffer))
			delete(ch.heightChan, ch.height)
			ch.height++
			ch.heightLock.Unlock()
			close(exitHeight)
			//Remove from
			//CLEANING
			for i := 0; i < ch.Total; i++ {
				close(rbc[i].exitRecv)
				close(aba[i].exitRecv)
			}

			logger.Debugf("ch.blockout")
			if len(ch.newBlockChan) == 0 {

				ch.bufferLock.Lock()
				if uint64(len(ch.txbuffer)) >= ch.batchSize {
					logger.Debugf("OUT PUSHING HEIGHT")
					ch.newBlockChan <- nil
				}
				ch.bufferLock.Unlock()
			}
		}

	}
}
