package honeybadgerbft

import (
	"crypto/rand"
	"testing"
	"time"
)

func randData(len int) []byte {
	data := make([]byte, len)
	// token := make([]byte, 4)
	rand.Read(data)
	return data
}

func TestOnlyOneRBCOneABA(t *testing.T) {
	// total := 1
	// tol := 1
	aba := make([]*BinaryAgreement, 1)
	aba[0] = &BinaryAgreement{In: make(chan bool), Out: make(chan bool)}
	go func() {
		data := <-aba[0].In
		aba[0].Out <- data
	}()
	rbc := make([]*ReliableBroadcast, 1)
	rbc[0] = &ReliableBroadcast{In: make(chan []byte), Out: make(chan []byte)}
	go func() {
		data := <-rbc[0].In
		rbc[0].Out <- data
	}()

	acs := NewCommonSubset(0, 1, 1, rbc, aba)

	acs.In <- randData(4)
	logger.Debugf("out:", <-acs.Out)
}

func TestWithNInstances(t *testing.T) {
	for k := 0; k < 10; k++ {
		total := 4
		// tol := 1
		aba := make([]*BinaryAgreement, total)
		rbc := make([]*ReliableBroadcast, total)

		for i := 0; i < total; i++ {
			aba[i] = &BinaryAgreement{In: make(chan bool), Out: make(chan bool)}

			go func(i int) {
				data := <-aba[i].In
				logger.Debugf("ABA IN : %v", data)
				aba[i].Out <- data
			}(i)

			rbc[i] = &ReliableBroadcast{In: make(chan []byte), Out: make(chan []byte)}
			go func(i int) {
				var data []byte
				if i == 0 {
					data = <-rbc[i].In
				} else {
					data = randData(40)
				}
				if i != 3 {
					rbc[i].Out <- data
				}
			}(i)
		}

		acs := NewCommonSubset(0, 4, 1, rbc, aba)

		acs.In <- randData(4)
		time.Sleep(2 * time.Second)
		// data := <-aba[3].In
		// aba[3].Out <- data
		logger.Debugf("out:", <-acs.Out)
	}
}
