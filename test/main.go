package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	cb "github.com/hyperledger/fabric/protos/common"

	honeybadgerbft "github.com/yadavdeepak95/HoneyBadgerBFT"
)

type Config struct {
	ConnectionList []string `json:"connection"`
	Total          int      `json:"total"`
	Tol            int      `json:"tol"`
	BatchSize      int      `json:"batchSize"`
	Index          int      `json:"index"`
	N              int      `json:"n"`
	Txn            int      `json:"txn"`
	Size           int      `json:"size"`
	Random         bool     `json:"random"`
	Fixed          bool     `json:"fixed"`
	FixedTime      int      `json:"fixedTime"`
}

func main() {

	ConnectionList, total, tol, batchSize, Index, n, txn, size, random, fixed, fixedTime := readfile()

	args := os.Args
	i, _ := (strconv.ParseInt(args[1], 10, 64))
	Index = int(i)

	// go http.ListenAndServe(":808"+string(Index), nil)
	// f, _ := os.Create("p" + args[1] + ".trace")
	// trace.Start(f)
	// defer trace.Stop()
	// fmt.Printf("%v", []string{":3000", ":3001", ":3002", "172.:3003"})

	// logging.LogLevel("INFO")

	chain := honeybadgerbft.NewWrapper(uint64(batchSize), ConnectionList, Index, total, tol, random, fixed, fixedTime)
	time.Sleep(2 * time.Second)
	startTime := time.Now()

	for j := 0; j < n; j++ {

		time.Sleep(1 * time.Second)
		for i := 0; i < txn; i++ {
			token := make([]byte, size)
			rand.Read(token)
			// fmt.Printf("\n\n%v::%v", i, token)
			chain.Enqueue(&cb.Envelope{
				Payload:   token,
				Signature: token,
			})
		}
	}
	fmt.Println("%v, %v", n*txn, time.Since(startTime))
}

func readfile() (ConnectionList []string, Total int, Tol int, BatchSize int, Index int, N int, Txn int, Size int, Random bool, Fixed bool, FixedTime int) {
	file, _ := ioutil.ReadFile("config.txt")
	data := Config{}
	json.Unmarshal([]byte(file), &data)
	ConnectionList = data.ConnectionList
	Total = data.Total
	Tol = data.Tol
	BatchSize = data.BatchSize
	Index = data.Index
	N = data.N
	Txn = data.Txn
	Size = data.Size
	Random = data.Random
	Fixed = data.Fixed
	FixedTime = data.FixedTime
	fmt.Printf("%+v", data)
	return
}
