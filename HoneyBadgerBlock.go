package honeybadgerbft

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/golang/protobuf/proto"

	cb "github.com/hyperledger/fabric/protos/common"

	ab "github.com/yadavdeepak95/HoneyBadgerBFT/proto/orderer"

	"github.com/hyperledger/fabric/protos/utils"
)

type HoneyBadgerBlock struct {
	total        int
	maxMalicious int
	channel      chan *ab.HoneyBadgerBFTMessage
	// broadcastFunc func(msg *ab.HoneyBadgerBFTMessageThresholdEncryption)
	acs *CommonSubset

	In  chan []*cb.Envelope
	Out chan []*cb.Envelope
}

// func NewHoneyBadgerBlock(total int, maxMalicious int, broadcastFunc func(msg ab.HoneyBadgerBFTMessage), acs *CommonSubset) (result *HoneyBadgerBlock) {
func NewHoneyBadgerBlock(total int, maxMalicious int, acs *CommonSubset) (result *HoneyBadgerBlock) {

	// bc := func(msg *ab.HoneyBadgerBFTMessageThresholdEncryption) {
	// 	broadcastFunc(ab.HoneyBadgerBFTMessage{ThresholdEncryption: msg})
	// }
	result = &HoneyBadgerBlock{
		total:        total,
		maxMalicious: maxMalicious,
		// broadcastFunc: bc,
		acs: acs,

		In:  make(chan []*cb.Envelope),
		Out: make(chan []*cb.Envelope),
	}
	go result.honeyBadgerBlockService()
	return result
}

func (block *HoneyBadgerBlock) honeyBadgerBlockService() {
	// TODO: check that propose_in is the correct length, not too large
	committingBatch := <-block.In
	logger.Debugf("BLOCK input: []*cb.Envelop(len=%v)", len(committingBatch))

	toACS, err := encodeTransactions(committingBatch)
	if err != nil {
		logger.Panic(err)
	}

	block.acs.In <- toACS

	fromACS := <-block.acs.Out
	if len(fromACS) != block.total {
		logger.Panicf("Wrong number of acs output")
	}
	count := 0
	for _, v := range fromACS {
		if v != nil {
			count++
		}
	}
	if count < block.total-block.maxMalicious {
		logger.Panicf("Wrong number of acs valid output")
	}

	var committedBatch []*cb.Envelope
	for _, v := range fromACS {
		if v == nil {
			continue
		}
		transations, err := decodeTransactions(v)
		if err != nil {
			logger.Panic(err)
		}
		committedBatch = append(committedBatch, transations...)

	}
	// logger.Debugf("output::::::::::::::::::::::::::::::%v", committedBatch)

	logger.Debugf("BLOCK output: []*cb.Envelop(len=%v)", len(committedBatch))
	block.Out <- committedBatch
}

//////////////////////////////////////////////////////////////////////////////////
//                                                                              //
//                                  TOOLS                                       //
//                                                                              //
//////////////////////////////////////////////////////////////////////////////////

func encodeByteArrays(arrays [][]byte) []byte {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, int32(len(arrays)))
	for _, array := range arrays {
		binary.Write(buf, binary.BigEndian, int32(len(array)))
		buf.Write(array)
	}
	return buf.Bytes()
}

func decodeByteArrays(data []byte) ([][]byte, error) {
	buf := bytes.NewBuffer(data)
	var l int32
	err := binary.Read(buf, binary.BigEndian, &l)
	if err != nil {
		return nil, fmt.Errorf("Error occured when decoding transactions: %s", err)
	}
	result := make([][]byte, l)
	for i := int32(0); i < l; i++ {
		var ll int32
		err := binary.Read(buf, binary.BigEndian, &ll)
		if err != nil {
			return nil, fmt.Errorf("Error occured when decoding transactions: %s", err)
		}
		array := make([]byte, ll)
		n, err := buf.Read(array)
		if err != nil {
			return nil, fmt.Errorf("Error occured when decoding transactions: %s", err)
		}
		if int32(n) != ll {
			return nil, fmt.Errorf("Error occured when decoding transactions: length mismatch")
		}
		result[i] = array
	}
	if len(buf.Bytes()) > 0 {
		return nil, fmt.Errorf("Error occured when decoding transactions: total length mismatch")
	}
	return result, nil
}

func encodeTransactions(transactions []*cb.Envelope) ([]byte, error) {
	var arrays = make([][]byte, len(transactions))
	for i, tx := range transactions {
		array, err := utils.Marshal(tx)
		arrays[i] = array
		if err != nil {
			return nil, err
		}
	}
	return encodeByteArrays(arrays), nil
}

func decodeTransactions(data []byte) ([]*cb.Envelope, error) {
	arrays, err := decodeByteArrays(data)
	if err != nil {
		return nil, err
	}
	var transactions = make([]*cb.Envelope, len(arrays))
	for i, array := range arrays {
		tx := new(cb.Envelope)
		err := proto.Unmarshal(array, tx)
		if err != nil {
			return nil, err
		}
		transactions[i] = tx
	}
	return transactions, nil
}
