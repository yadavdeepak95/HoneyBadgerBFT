package honeybadgerbft

import "sync"

type CommonSubset struct {
	ordererIndex int
	total        int
	tolerance    int
	rbc          []*ReliableBroadcast
	aba          []*BinaryAgreement
	lock         sync.Mutex
	In           chan []byte
	Out          chan [][]byte
}

func NewCommonSubset(ordererIndex int, total int, tolerance int, rbc []*ReliableBroadcast, aba []*BinaryAgreement) (result *CommonSubset) {
	result = &CommonSubset{
		ordererIndex: ordererIndex,
		total:        total,
		tolerance:    tolerance,
		rbc:          rbc,
		aba:          aba,

		In:  make(chan []byte),
		Out: make(chan [][]byte),
	}
	go result.commonSubsetService()
	return result
}

func (acs *CommonSubset) commonSubsetService() {

	data := <-acs.In
	acs.rbc[acs.ordererIndex].In <- data

	var abaInputted = make(map[int]bool)
	var joinABA = make(chan interface{})
	var abaOutputs = make([]bool, acs.total)

	var joinRBC = make([]chan interface{}, acs.total)
	var killRBC = make([]chan interface{}, acs.total)
	var rbcOutputs = make([][]byte, acs.total)

	receiveRBC := func(instanceIndex int) {
		select {
		case <-killRBC[instanceIndex]:
			logger.Debugf("killed RBC[%v]", instanceIndex)
			return
		case data := <-acs.rbc[instanceIndex].Out:
			acs.lock.Lock()
			rbcOutputs[instanceIndex] = data
			if !abaInputted[instanceIndex] {
				acs.aba[instanceIndex].In <- true
				abaInputted[instanceIndex] = true
			}
			acs.lock.Unlock()
			joinRBC[instanceIndex] <- nil
		}
	}

	for index := range acs.rbc {
		joinRBC[index] = make(chan interface{}, 2)
		killRBC[index] = make(chan interface{}, 2)
	}
	for index := range acs.rbc {
		go receiveRBC(index)
	}

	receiveABA := func(instanceIndex int) {
		data := <-acs.aba[instanceIndex].Out
		acs.lock.Lock()
		abaOutputs[instanceIndex] = data
		sum := 0
		for _, v := range abaOutputs {
			if v {
				sum++
			}
		}
		if sum >= acs.total-acs.tolerance {
			for index, aba := range acs.aba {
				if !abaInputted[index] {
					aba.In <- false
					logger.Debugf("ACS putting false as no RBC")
					abaInputted[index] = true
				}
			}
		}
		acs.lock.Unlock()
		joinABA <- nil
	}
	for index := range acs.aba {
		go receiveABA(index)
	}
	for _, _ = range acs.aba {
		<-joinABA
		// close(acsInstance.exitRecv)
	}
	logger.Debugf("EVERY BODY JOINED")

	for index, abaOutput := range abaOutputs {
		//

		if abaOutput {

			<-joinRBC[index]
			// if rbcOutputs[index] == nil {
			// 	logger.Panicf("RBC output of %v instance is nil", index)
			// }
		} else {

			// <-joinRBC[index]
			killRBC[index] <- nil
			logger.Debugf("kill signal sent RBC[%v]", index)
			rbcOutputs[index] = nil
		}
	}
	var avaliableCount int
	for _, v := range rbcOutputs {
		if v != nil {
			avaliableCount++
			logger.Debugf("OUT::  %v", v)
		}

	}
	logger.Debugf("ACS output: [][]byte(len=%v,aval=%v)", len(rbcOutputs), avaliableCount)
	acs.Out <- rbcOutputs

}
