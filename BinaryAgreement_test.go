package honeybadgerbft

import (
	"fmt"
	"testing"
	"time"

	ab "github.com/hyperledger/fabric/protos/orderer"
)

func sendEstMessages(in chan *ab.HoneyBadgerBFTMessage, round uint64, value bool) {
	for i := 0; i < 4; i++ {
		// for j := 0; j < 2; j++ {
		in <- &ab.HoneyBadgerBFTMessage{
			Sender: uint64(i),
			BinaryAgreement: &ab.HoneyBadgerBFTMessageBinaryAgreement{
				Est:   &ab.HoneyBadgerBFTMessageBinaryAgreementEST{},
				Value: value,
				Round: round,
			},
		}
		// }
	}
	// for i := 0; i < 4; i++ {
	// 	for j := 0; j < 2; j++ {
	// 		in <- &ab.HoneyBadgerBFTMessage{
	// 			Sender: uint64(i),
	// 			BinaryAgreement: &ab.HoneyBadgerBFTMessageBinaryAgreement{
	// 				Est:   &ab.HoneyBadgerBFTMessageBinaryAgreementEST{},
	// 				Value: (j == 0),
	// 				Round: 0,
	// 			},
	// 		}
	// 	}
	// }
	in <- &ab.HoneyBadgerBFTMessage{}
}

func sendAuxMessages(in chan *ab.HoneyBadgerBFTMessage, round uint64, value bool) {
	for i := 0; i < 4; i++ {
		// for j := 0; j < 2; j++ {
		in <- &ab.HoneyBadgerBFTMessage{
			Sender: uint64(i),
			BinaryAgreement: &ab.HoneyBadgerBFTMessageBinaryAgreement{
				Aux:   &ab.HoneyBadgerBFTMessageBinaryAgreementAUX{},
				Value: value,
				Round: round,
			},
		}
		// }
	}
	in <- &ab.HoneyBadgerBFTMessage{}

}

// func TestEstMapLogic(t *testing.T) {

// 	total := 4
// 	// for In should recv est for broadcast
// 	sendFunc := func(index int, msg ab.HoneyBadgerBFTMessage) {
// 		msg.Receiver = uint64(index)
// 		// abaInstanceRecvMsgChannels[msg.Receiver] <- &msg
// 		logger.Debugf("%v", msg)
// 	}
// 	broadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
// 		for i := 0; i < total; i++ {
// 			sendFunc(i, msg)
// 		}
// 	}

// 	componentBroadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
// 		broadcastFunc(msg)
// 	}
// 	inchannel := make(chan *ab.HoneyBadgerBFTMessage, 66666)
// 	aba := NewBinaryAgreement(0, 4, 1, inchannel, componentBroadcastFunc)
// 	// aba.In <- true
// 	sendEstMessages(aba.inchannel, 0, true)
// 	sendEstMessages(aba.inchannel, 0, false)
// 	sendEstMessages(aba.inchannel, 1, false)
// 	sendAuxMessages(aba.inchannel, 1, false)
// 	sendAuxMessages(aba.inchannel, 1, false)
// 	sendAuxMessages(aba.inchannel, 0, false)
// 	// logger.Debugf("Sending AUX")

// 	logger.Debugf("%v", <-aba.Out)
// 	time.Sleep(2 * time.Second)
// }

func TestOutProblem(t *testing.T) {
	total := 4
	tol := 1
	i := 0
	var abaInstanceRecvMsgChannels = make([]chan *ab.HoneyBadgerBFTMessage, total)

	sendFunc := func(index int, msg ab.HoneyBadgerBFTMessage) {
		msg.Receiver = uint64(index)
		abaInstanceRecvMsgChannels[msg.Receiver] <- &msg
	}
	broadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
		for i := 0; i < total; i++ {
			sendFunc(i, msg)
		}
	}

	componentBroadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
		broadcastFunc(msg)
	}

	for i := 0; i < total; i++ {
		abaInstanceRecvMsgChannels[i] = make(chan *ab.HoneyBadgerBFTMessage, 666666)
	}

	aba := NewBinaryAgreement((i), (total), (tol), abaInstanceRecvMsgChannels[i], componentBroadcastFunc) // TODO: a better way to eval whether i is leader, ListenAddress may difference with address in connectionlist
	go func() {
		aba.Out <- true
	}()
	close(aba.exitRecv)
	<-aba.Out
}

func TestForOneinstance(t *testing.T) {
	total := 4
	tol := 1
	j := 0
	// for j := 0; j < 100; j++ {

	var abaInstanceRecvMsgChannels = make([]chan *ab.HoneyBadgerBFTMessage, total)

	sendFunc := func(index int, msg ab.HoneyBadgerBFTMessage) {
		msg.Receiver = uint64(index)
		abaInstanceRecvMsgChannels[msg.Receiver] <- &msg
	}
	broadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
		for i := 0; i < total; i++ {
			sendFunc(i, msg)
		}
	}

	componentBroadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
		broadcastFunc(msg)
	}

	for i := 0; i < total; i++ {
		abaInstanceRecvMsgChannels[i] = make(chan *ab.HoneyBadgerBFTMessage, 666666)
	}

	var aba = make([]*BinaryAgreement, total)
	ch := make(chan bool, 4)
	for i := 0; i < total; i++ {
		aba[i] = NewBinaryAgreement((i), (total), (tol), abaInstanceRecvMsgChannels[i], componentBroadcastFunc) // TODO: a better way to eval whether i is leader, ListenAddress may difference with address in connectionlist
		logger.Debugf("stuck here")

	}
	go func(ch chan bool) {
		for i := 0; i < total; i++ {
			<-aba[i].Out
			// ch <- temp
			// close(aba[i].exitRecv)
		}
	}(ch)

	for i := 0; i < total; i++ {
		aba[i].In <- (j%2 == 0)
		// <-ch
		// // close(aba[i].exitRecv)
	}
	logger.Debugf("\n\n\ni = %v\nin:%v\n\n\n", j, (j%2 == 0))
	// abaInstanceRecvMsgChannels = nil
	// }
	time.Sleep(2 * time.Second)
}

func TestForOneinstanceOnlyCorrect(t *testing.T) {
	total := 4
	tol := 1
	j := 0
	// for j := 0; j < 100; j++ {
	var abaInstanceRecvMsgChannels = make([]chan *ab.HoneyBadgerBFTMessage, total)

	sendFunc := func(index int, msg ab.HoneyBadgerBFTMessage) {
		msg.Receiver = uint64(index)
		abaInstanceRecvMsgChannels[msg.Receiver] <- &msg
	}
	broadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
		for i := 0; i < total; i++ {
			sendFunc(i, msg)
		}
	}

	componentBroadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
		broadcastFunc(msg)
	}

	for i := 0; i < total; i++ {
		abaInstanceRecvMsgChannels[i] = make(chan *ab.HoneyBadgerBFTMessage, 666666)
	}

	var aba = make([]*BinaryAgreement, total)
	ch := make(chan bool, 3)
	go func(ch chan bool) {
		for i := 0; i < total-tol; i++ {
			temp := <-aba[i].Out
			ch <- temp
			// close(aba[i].exitRecv)
		}
	}(ch)
	for i := 0; i < total-tol; i++ {
		aba[i] = NewBinaryAgreement((i), (total), (tol), abaInstanceRecvMsgChannels[i], componentBroadcastFunc) // TODO: a better way to eval whether i is leader, ListenAddress may difference with address in connectionlist
		aba[i].In <- (j%2 == 0)
	}
	for i := 0; i < total-tol; i++ {
		<-ch
		// close(aba[i].exitRecv)
	}
	fmt.Printf("\n\n\ni = %v\nin:%v\n\n\n", j, (j%2 == 0))
	// abaInstanceRecvMsgChannels = nil
	// }
}

func TestForOneinstanceAlternateValues(t *testing.T) {
	total := 4
	tol := 1

	// for j := 0; j < 10; j++ {
	var abaInstanceRecvMsgChannels = make([]chan *ab.HoneyBadgerBFTMessage, total)

	sendFunc := func(index int, msg ab.HoneyBadgerBFTMessage) {
		msg.Receiver = uint64(index)
		abaInstanceRecvMsgChannels[msg.Receiver] <- &msg
	}
	broadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
		for i := 0; i < total; i++ {
			sendFunc(i, msg)
		}
	}

	componentBroadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
		broadcastFunc(msg)
	}

	for i := 0; i < total; i++ {
		abaInstanceRecvMsgChannels[i] = make(chan *ab.HoneyBadgerBFTMessage, 666666)
	}

	var aba = make([]*BinaryAgreement, total)

	ch := make(chan bool, 4)

	for i := 0; i < total; i++ {
		aba[i] = NewBinaryAgreement((i), (total), (tol), abaInstanceRecvMsgChannels[i], componentBroadcastFunc) // TODO: a better way to eval whether i is leader, ListenAddress may difference with address in connectionlist
	}
	for i := 0; i < total; i++ {
		aba[i].In <- (i%2 == 0)
	}

	go func(ch chan bool) {
		for i := 0; i < total; i++ {
			temp := <-aba[i].Out
			ch <- temp
			// close(aba[i].exitRecv)
		}
	}(ch)
	for i := 0; i < total; i++ {
		<-ch

		// close(aba[i].exitRecv)
	}
	// fmt.Printf("\n\n\ni = %v\nin:%v\n\n\n", j, (j%2 == 0))
	// }
}
func TestTimeWaste(t *testing.T) {
	time.Sleep(5 * time.Second)
}
