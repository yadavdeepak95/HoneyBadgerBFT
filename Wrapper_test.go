package honeybadgerbft

import (
	"testing"
	"time"
)

func TestInTxns(t *testing.T) {
	// msgChan := &MessageChannels{In: make(chan interface{}, 66666), Out: make(chan interface{}, 666666)}
	// // msgChan.In <- &ab.HoneyBadgerBFTMessage{}
	// msgChan.In <- &cb.Envelope{}
	// msgChan.In <- &cb.Envelope{}
	// msgChan.In <- &ab.HoneyBadgerBFTMessage{Height: 1}

	// msgChan.In <- &ab.HoneyBadgerBFTMessage{Height: 0}
	// msgChan.In <- &ab.HoneyBadgerBFTMessage{Height: 0}
	// msgChan.In <- &ab.HoneyBadgerBFTMessage{Height: 0}

	// msgChan.In <- &ab.HoneyBadgerBFTMessage{Height: 1}

	// // msgChan.In <- true
	// ch := NewWrapper(2, msgChan)
	// ch.txbuffer = []*cb.Envelope{}
	// // logger.Debugf("%v", <-ch.newBlockChan)
	// time.Sleep(5 * time.Second)

	// ch.blockOut <- nil
	// // time.Sleep(20 * time.Second)

	// msgChan.In <- &cb.Envelope{}
	// msgChan.In <- &cb.Envelope{}
	// msgChan.In <- &ab.HoneyBadgerBFTMessage{Height: 0}

	// msgChan.In <- &ab.HoneyBadgerBFTMessage{Height: 1}
	// time.Sleep(5 * time.Second)

	// ch.blockOut <- nil
	// // logger.Debugf("%v", <-ch.newBlockChan)
	// ch.blockOut <- nil

	// logger.Debugf("%v", <-ch.newBlockChan)

	time.Sleep(5 * time.Second)
}

// func TestHeight(t *testing.T) {
// 	msgChan := &MessageChannels{In: make(chan interface{}, 66666), Out: make(chan interface{}, 666666)}

// 	msgChan.In <- &cb.Envelope{}
// 	msgChan.In <- &cb.Envelope{}
// 	msgChan.In <- &ab.HoneyBadgerBFTMessage{Height: 0}

// 	msgChan.In <- &ab.HoneyBadgerBFTMessage{Height: 2}

// 	ch := NewWrapper(2, msgChan)
// 	ch.txbuffer = []*cb.Envelope{}

// 	ch.blockOut <- nil
// 	time.Sleep(20 * time.Second)
// 	msgChan.In <- &ab.HoneyBadgerBFTMessage{Height: 0}

// 	msgChan.In <- &ab.HoneyBadgerBFTMessage{Height: 2}

// }
