package honeybadgerbft

import (
	"sync"

	ab "github.com/yadavdeepak95/HoneyBadgerBFT/proto/orderer"
)

type BinaryAgreement struct {
	total         int
	tolerance     int
	broadcastFunc func(msg *ab.HoneyBadgerBFTMessageBinaryAgreement)
	inchannel     chan *ab.HoneyBadgerBFTMessage
	instanceIndex int
	//Round  value
	estRecv        map[uint64]map[bool]map[uint64]bool
	estSent        map[uint64]map[bool]bool
	auxValues      map[uint64]map[bool]map[uint64]bool
	binValues      map[uint64]map[bool]bool
	epoch          uint64
	alreadyDecided *bool
	Lock           sync.Mutex
	futureChan     []*ab.HoneyBadgerBFTMessage
	auxChan        map[uint64]chan bool
	estChan        map[uint64]chan bool
	exitRecv       chan interface{}
	In             chan bool
	Out            chan bool
}

func NewBinaryAgreement(instanceIndex int, total, tolerance int, inchannel chan *ab.HoneyBadgerBFTMessage, broadcastFunc func(msg ab.HoneyBadgerBFTMessage)) *BinaryAgreement {
	bc := func(msg *ab.HoneyBadgerBFTMessageBinaryAgreement) {
		broadcastFunc(ab.HoneyBadgerBFTMessage{Sender: uint64(instanceIndex), BinaryAgreement: msg})
	}
	result := &BinaryAgreement{
		total:         total,
		tolerance:     tolerance,
		broadcastFunc: bc,
		instanceIndex: instanceIndex,
		inchannel:     inchannel,
		auxChan:       make(map[uint64](chan bool)),
		estChan:       make(map[uint64](chan bool)),
		In:            make(chan bool),
		Out:           make(chan bool),
		exitRecv:      make(chan interface{}),
	}
	go result.recvMsgs()
	return result
}

func (aba *BinaryAgreement) recvMsgs() {
	for {
		select {
		case <-aba.exitRecv:
			logger.Debugf("ABA[%v] receive service exit", aba.instanceIndex)
			return
		case msg := <-aba.In:
			// logger.Debug("got msg")
			aba.handleIn(msg)
		case msg := <-aba.inchannel:
			aba.handleMsg(msg)
		}
	}
}

func (aba *BinaryAgreement) handleIn(msg bool) {
	aba.Lock.Lock()
	defer aba.Lock.Unlock()
	if aba.epoch != 0 {
		return
	}
	// logger.Debugf("broacasting")
	if !aba.inEstSentSet(aba.epoch, msg) {

		aba.broadcastFunc(&ab.HoneyBadgerBFTMessageBinaryAgreement{
			Round: aba.epoch,
			Value: msg,
			Est:   &ab.HoneyBadgerBFTMessageBinaryAgreementEST{},
		})
		logger.Debugf("ABA[%v] : R[%v] : V[%v]---BROADCASTED EST FROM IN", aba.instanceIndex, aba.epoch, msg)

		aba.putEstSentSet(aba.epoch, msg)
	}
}
func (aba *BinaryAgreement) handleMsg(msg *ab.HoneyBadgerBFTMessage) {
	sender := msg.Sender
	ba := msg.GetBinaryAgreement()
	round := ba.GetRound()
	value := ba.GetValue()
	aba.Lock.Lock()
	defer aba.Lock.Unlock()
	if aba.alreadyDecided != nil {
		return
	}
	// logger.Debugf("ABA[%v] : R[%v] : V[%v] : S[%v] : EST[%v] : AUX[%v] -- INPUT", aba.instanceIndex, round, value, sender, ba.GetEst() != nil, ba.GetAux() != nil)
	if round < aba.epoch {
		logger.Debugf("ABA[%v] : R[%v] : V[%v] : S[%v] : T[est] -- OLD DROPPING", aba.instanceIndex, round, value, sender)
		return
	} else if round > aba.epoch {
		logger.Debugf("ABA[%v] : R[%v] : V[%v] : S[%v] : T[est] -- FUTURE ADDING TO QUEUE", aba.instanceIndex, round, value, sender)
		aba.futureChan = append(aba.futureChan, msg)
		return
	} else if ba.GetEst() != nil {

		//check for redundent messages
		if aba.inEstRecv(round, value, sender) {
			logger.Debugf("ABA[%v] : R[%v] : V[%v] : S[%v] : T[est] -- Redundent", aba.instanceIndex, round, value, sender)
			return
		}
		logger.Debugf("ABA[%v] : R[%v] : V[%v] : S[%v] : T[est]", aba.instanceIndex, round, value, sender)

		aba.putEstRecv(round, value, sender)
		if aba.lenEstRecvSet(round, value) >= uint64(aba.tolerance+1) && !aba.inEstSentSet(round, value) {

			aba.broadcastFunc(&ab.HoneyBadgerBFTMessageBinaryAgreement{
				Round: round,
				Value: value,
				Est:   &ab.HoneyBadgerBFTMessageBinaryAgreementEST{},
			})
			logger.Debugf("ABA[%v] : R[%v] : V[%v]---BROADCASTED EST", aba.instanceIndex, round, value)

			aba.putEstSentSet(round, value)

		}
		if aba.lenEstRecvSet(round, value) >= uint64(2*aba.tolerance+1) {
			logger.Debugf("ABA[%v] : R[%v] : V[%v]---Putting Bin Value", aba.instanceIndex, round, value)
			if aba.lenBinValuesSet(round) == 0 {
				aba.broadcastFunc(&ab.HoneyBadgerBFTMessageBinaryAgreement{
					Round: round,
					Value: value,
					Aux:   &ab.HoneyBadgerBFTMessageBinaryAgreementAUX{},
				})
			}
			aba.putBinValuesSet(round, value)
			aba.tryOutput()

		}
		// aba.Lock.Unlock()

	} else if ba.GetAux() != nil {

		if aba.inAuxValuesSet(round, value, sender) {
			logger.Debugf("ABA[%v] : R[%v] : V[%v] : S[%v] : T[Aux] -- Redundent", aba.instanceIndex, round, value, sender)

			return
		}
		aba.putAuxValuesSet(round, value, sender)
		aba.tryOutput()
	} else {
		// aba.Lock.Lock()
		logger.Debugf("%+v", aba.estRecv)
		// aba.Lock.Unlock()
	}
}

func (aba *BinaryAgreement) tryOutput() {

	var values = aba.listBinValuesSet(aba.epoch)
	if len(values) == 0 {
		return
	}
	var sum uint64
	for _, v := range values {
		sum += aba.lenAuxValuesSet(aba.epoch, v)
	}
	if sum < uint64(aba.total-aba.tolerance) {
		return
	}

	s := ((aba.epoch)%2 == 0)
	var est bool
	if len(values) == 1 {

		v := values[0]
		logger.Debugf("ABA[%v] Round[%v] est[%v] Coin[%v]", aba.instanceIndex, aba.epoch, v, s)
		if v == s {
			if aba.alreadyDecided == nil {
				aba.alreadyDecided = &v
				logger.Debugf("ABA[%v] <- Out", aba.instanceIndex)
				//TODO : Add function if found appropriate
				aba.Out <- v
				// close(aba.exitRecv)
			} else if *aba.alreadyDecided == v {
				return
			}
			est = v
		}
	} else {
		logger.Debugf("ABA[%v], Est[%v] ROUND[%v]", aba.instanceIndex, s, aba.epoch)
		est = s
	}
	aba.epoch++
	if !aba.inEstSentSet(aba.epoch, est) {
		logger.Debugf("ABA[%v] [%v][%v]---BROADCASTED EST", aba.instanceIndex, aba.epoch, est)
		aba.broadcastFunc(&ab.HoneyBadgerBFTMessageBinaryAgreement{
			Round: aba.epoch,
			Value: est,
			Est:   &ab.HoneyBadgerBFTMessageBinaryAgreementEST{},
		})
		aba.putEstSentSet(aba.epoch, est)
	}
	//add delayed msgs
	for _, msg := range aba.futureChan {
		aba.inchannel <- msg
	}
	aba.futureChan = []*ab.HoneyBadgerBFTMessage{}
}

func (aba *BinaryAgreement) putEstRecv(round uint64, value bool, sender uint64) {
	if aba.estRecv == nil {
		aba.estRecv = make(map[uint64]map[bool]map[uint64]bool)
	}
	if _, rE := aba.estRecv[round]; !rE {
		aba.estRecv[round] = make(map[bool]map[uint64]bool)
	}
	if _, rV := aba.estRecv[round][value]; !rV {
		aba.estRecv[round][value] = make(map[uint64]bool)
	}
	aba.estRecv[round][value][sender] = true
}

func (aba *BinaryAgreement) inEstRecv(round uint64, value bool, sender uint64) bool {

	if aba.estRecv != nil {
		if _, rE := aba.estRecv[round]; rE {
			if _, vE := aba.estRecv[round][value]; vE {
				if _, sE := aba.estRecv[round][value][sender]; sE {
					return true //aba.estRecv[round][value][sender]
				}
			}
		}
	}
	return false
}

func (aba *BinaryAgreement) lenEstRecvSet(round uint64, value bool) uint64 {
	// aba.estRecvLock.Lock()
	// defer aba.estRecvLock.Unlock()
	if aba.estRecv != nil {
		if _, rE := aba.estRecv[round]; rE {
			if _, vE := aba.estRecv[round][value]; vE {
				return uint64(len(aba.estRecv[round][value]))
			}
		}
	}
	return 0
}

func (aba *BinaryAgreement) putEstSentSet(round uint64, value bool) {
	// aba.estSentLock.Lock()
	// defer aba.estSentLock.Unlock()
	if aba.estSent == nil {
		aba.estSent = make(map[uint64]map[bool]bool)
	}
	if _, rE := aba.estSent[round]; !rE {
		aba.estSent[round] = make(map[bool]bool)
	}
	aba.estSent[round][value] = true
}

func (aba *BinaryAgreement) inEstSentSet(round uint64, value bool) bool {
	// aba.estSentLock.Lock()
	// defer aba.estSentLock.Unlock()
	if aba.estSent != nil {
		if _, rE := aba.estSent[round]; rE {
			if _, rV := aba.estSent[round][value]; rV {
				return true //aba.estSent[round][value]
			}
		}
	}
	return false
}

func (aba *BinaryAgreement) putBinValuesSet(round uint64, value bool) {
	// aba.binValuesLock.Lock()
	// defer aba.binValuesLock.Unlock()
	if aba.binValues == nil {
		aba.binValues = make(map[uint64]map[bool]bool)
	}
	if _, rE := aba.binValues[round]; !rE {
		aba.binValues[round] = make(map[bool]bool)
	}
	aba.binValues[round][value] = true
}

func (aba *BinaryAgreement) listBinValuesSet(round uint64) []bool {
	// aba.binValuesLock.Lock()
	// defer aba.binValuesLock.Unlock()
	result := []bool{}
	if aba.binValues != nil {
		if _, rE := aba.binValues[round]; rE {
			for v, _ := range aba.binValues[round] {
				result = append(result, v)

			}
		}
	}
	return result
}

func (aba *BinaryAgreement) lenBinValuesSet(round uint64) int {
	return len(aba.listBinValuesSet(round))
}

func (aba *BinaryAgreement) putAuxValuesSet(round uint64, value bool, sender uint64) {
	// aba.auxValuesLock.Lock()
	// defer aba.auxValuesLock.Unlock()
	if aba.auxValues == nil {
		aba.auxValues = make(map[uint64]map[bool]map[uint64]bool)
	}
	if _, rE := aba.auxValues[round]; !rE {
		aba.auxValues[round] = make(map[bool]map[uint64]bool)
	}
	if _, rV := aba.auxValues[round][value]; !rV {
		aba.auxValues[round][value] = make(map[uint64]bool)
	}
	aba.auxValues[round][value][sender] = true
}

func (aba *BinaryAgreement) inAuxValuesSet(round uint64, value bool, sender uint64) bool {
	// aba.auxValuesLock.Lock()
	// defer aba.auxValuesLock.Unlock()
	if aba.auxValues != nil {
		if _, rE := aba.auxValues[round]; rE {
			if _, vE := aba.auxValues[round][value]; vE {
				if _, sE := aba.auxValues[round][value][sender]; sE {
					return true // aba.auxValues[round][value][sender]
				}
			}
		}
	}
	return false
}

func (aba *BinaryAgreement) lenAuxValuesSet(round uint64, value bool) uint64 {
	// aba.auxValuesLock.Lock()
	// defer aba.auxValuesLock.Unlock()
	if aba.auxValues != nil {
		if _, rE := aba.auxValues[round]; rE {
			if _, vE := aba.auxValues[round][value]; vE {
				return uint64(len(aba.auxValues[round][value]))
			}
		}
	}
	return 0
}
