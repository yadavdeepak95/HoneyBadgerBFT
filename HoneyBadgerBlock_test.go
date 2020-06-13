package honeybadgerbft

import (
	"crypto/rand"
	"testing"

	cb "github.com/hyperledger/fabric/protos/common"
)

func TestInOut(t *testing.T) {
	acs := &CommonSubset{In: make(chan []byte), Out: make(chan [][]byte)}
	block := NewHoneyBadgerBlock(1, 1, acs)
	in := []*cb.Envelope{}
	for i := 0; i < 4; i++ {
		token := make([]byte, 4)
		rand.Read(token)

		in = append(in, &cb.Envelope{
			Payload:   token,
			Signature: token,
		})
	}

	block.In <- in
	// encodeTransactions(in)
	temp := make([][]byte, 1)

	temp[0] = <-acs.In
	acs.Out <- temp

	out := <-block.Out
	logger.Debugf("IN [%v]", in)
	logger.Debugf("OUT[%v]", out)
}
