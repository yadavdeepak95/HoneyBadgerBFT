package honeybadgerbft

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	ab "github.com/hyperledger/fabric/protos/orderer"
	"github.com/klauspost/reedsolomon"
)

func TestReedSolman(t *testing.T) {
	total := 64
	tol := 4
	data := make([]byte, 400)
	// token := make([]byte, 4)
	rand.Read(data)
	logger.Debugf("IN data : %v", data)
	var K = total - 2*tol
	enc, _ := reedsolomon.New(K, total-K)
	blocks, _ := Encode(enc, data)
	logger.Debugf("Encoded : %v", blocks)
	enc.Reconstruct(blocks)
	logger.Debugf("Decoded : %v", blocks)
	var value []byte
	for _, data := range blocks[:K] {
		logger.Debugf("%v", data)
		value = append(value, data...)
	}
	logger.Debugf("Out data : %v", value[:400])
	logger.Debugf("compare : %v", bytes.Compare(data, value[:400]))

}

func TestIntialization(t *testing.T) {

	sendFunc := func(index int, msg ab.HoneyBadgerBFTMessage) {
	}
	broadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {

	}
	rbcInstanceRecvMsgChannels := make(chan *ab.HoneyBadgerBFTMessage, 666666)
	instanceIndex := 0 // NOTE important to copy i
	componentSendFunc := func(index int, msg ab.HoneyBadgerBFTMessage) {
		msg.Instance = uint64(instanceIndex)
		sendFunc(index, msg)
	}
	componentBroadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
		msg.Instance = uint64(instanceIndex)
		broadcastFunc(msg)
	}
	rbc := NewReliableBroadcast(0, 4, 1, 0, 0, rbcInstanceRecvMsgChannels, componentSendFunc, componentBroadcastFunc) // TODO: a better way to eval whether i is leader, ListenAddress may difference with address in connectionlist

	if rbc != nil {
		fmt.Printf("Success")
	} else {
		fmt.Errorf("Fail")
	}
}

func TestEncodingandBroadcast(t *testing.T) {

	sendFunc := func(index int, msg ab.HoneyBadgerBFTMessage) {
		msg.Receiver = uint64(index)
		// fmt.Printf("\n\n%+v\n\n", msg)
	}
	broadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
		fmt.Printf("%+v", msg)
	}
	rbcInstanceRecvMsgChannels := make(chan *ab.HoneyBadgerBFTMessage, 666666)
	instanceIndex := 0 // NOTE important to copy i
	componentSendFunc := func(index int, msg ab.HoneyBadgerBFTMessage) {
		msg.Instance = uint64(instanceIndex)
		sendFunc(index, msg)
	}
	componentBroadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
		msg.Instance = uint64(instanceIndex)
		broadcastFunc(msg)
	}
	rbc := NewReliableBroadcast(0, 7, 2, 0, 0, rbcInstanceRecvMsgChannels, componentSendFunc, componentBroadcastFunc) // TODO: a better way to eval whether i is leader, ListenAddress may difference with address in connectionlist

	if rbc != nil {
		fmt.Printf("Intialization Success")
		token := make([]byte, 4)
		rand.Read(token)
		rbc.In <- token
		// <-rbc.Out
	} else {
		fmt.Errorf("Fail")
	}
}

func TestOneInstanceRBCChannel(t *testing.T) {

	//Channels for message passing
	total := 64
	tol := 4
	// for i := 0; i < 40; i++ {
	var rbcInstanceRecvMsgChannels = make([]chan *ab.HoneyBadgerBFTMessage, total)

	sendFunc := func(index int, msg ab.HoneyBadgerBFTMessage) {
		msg.Receiver = uint64(index)
		// fmt.Printf("\n\n%+v\n\n", msg)
		rbcInstanceRecvMsgChannels[msg.Receiver] <- &msg
	}
	broadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
		for i := 0; i < total; i++ {
			sendFunc(i, msg)
		}
	}
	componentSendFunc := func(index int, msg ab.HoneyBadgerBFTMessage) {
		// msg.Instance = uint64(msg.Receiver)
		sendFunc(index, msg)
	}
	componentBroadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
		// msg.Instance = uint64(msg.Receiver)
		broadcastFunc(msg)
	}
	for i := 0; i < total; i++ {
		rbcInstanceRecvMsgChannels[i] = make(chan *ab.HoneyBadgerBFTMessage, 666666)
	}
	var rbc = make([]*ReliableBroadcast, total)
	for i := 0; i < total; i++ {
		rbc[i] = NewReliableBroadcast(0, total, tol, i, 0, rbcInstanceRecvMsgChannels[i], componentSendFunc, componentBroadcastFunc) // TODO: a better way to eval whether i is leader, ListenAddress may difference with address in connectionlist
	}
	token := make([]byte, 4)
	rand.Read(token)
	rbc[0].In <- token
	out := <-rbc[0].Out
	fmt.Printf("in:%v\nout:%v,%v", token, out, token)
	time.Sleep(2 * time.Second)
	// }

}

// func TestFaultyOneInstanceRBCChannel(t *testing.T) {

// 	//Channels for message passing
// 	// for i := 0; i < 40; i++ {
// 	var rbcInstanceRecvMsgChannels = make([]chan *ab.HoneyBadgerBFTMessage, 4)

// 	sendFunc := func(index int, msg ab.HoneyBadgerBFTMessage) {
// 		msg.Receiver = uint64(index)
// 		// fmt.Printf("\n\n%+v\n\n", msg)
// 		if msg.Sender != uint64(3) {
// 			rbcInstanceRecvMsgChannels[msg.Receiver] <- &msg
// 		}
// 	}
// 	broadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
// 		for i := 0; i < 4; i++ {
// 			sendFunc(i, msg)
// 		}
// 	}
// 	componentSendFunc := func(index int, msg ab.HoneyBadgerBFTMessage) {
// 		// msg.Instance = uint64(msg.Receiver)
// 		sendFunc(index, msg)
// 	}
// 	componentBroadcastFunc := func(msg ab.HoneyBadgerBFTMessage) {
// 		// msg.Instance = uint64(msg.Receiver)
// 		broadcastFunc(msg)
// 	}
// 	for i := 0; i < 4; i++ {
// 		rbcInstanceRecvMsgChannels[i] = make(chan *ab.HoneyBadgerBFTMessage, 666666)
// 	}
// 	var rbc = make([]*ReliableBroadcast, 4)
// 	for i := 0; i < 4; i++ {
// 		rbc[i] = NewReliableBroadcast(0, 4, 1, i, 0, rbcInstanceRecvMsgChannels[i], componentSendFunc, componentBroadcastFunc) // TODO: a better way to eval whether i is leader, ListenAddress may difference with address in connectionlist
// 	}
// 	token := make([]byte, 4)
// 	rand.Read(token)
// 	rbc[0].In <- token
// 	out := <-rbc[0].Out
// 	<-rbc[1].Out
// 	<-rbc[2].Out
// 	<-rbc[3].Out
// 	fmt.Printf("in:%v\nout:%v", token, out)
// 	// }

// }

// func TestFaultyNETOneInstanceRBCChannel(t *testing.T) {

// }
