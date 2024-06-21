package main

import (
  "math/big"
  "fmt"
  "encoding/hex"
  "crypto/sha256"
  _ "bytes"
)

/*
Transaction   Contract       Size
-------------------------------------------
sender        sender       20 bytes
recipient     0x0          20 bytes
value         endowment     4 bytes (uint32)
fee           fee           4 bytes (uint32)
d_size        o_size        4 bytes (uint32)
data          ops           *
signature     signature    64 bytes
*/

var StepFee     *big.Int = new(big.Int)
var TxFee       *big.Int = new(big.Int)
var ContractFee *big.Int = new(big.Int)
var MemFee      *big.Int = new(big.Int)
var DataFee     *big.Int = new(big.Int)
var CryptoFee   *big.Int = new(big.Int)
var ExtroFee    *big.Int = new(big.Int)

var Period1Reward *big.Int = new(big.Int)
var Period2Reward *big.Int = new(big.Int)
var Period3Reward *big.Int = new(big.Int)
var Period4Reward *big.Int = new(big.Int)

type Transaction struct {
  RlpSerializer

  sender      string
  recipient   uint32
  value       uint32
  fee         uint32
  data        []string
  memory      []int

  // To be removed
  signature   string
  addr        string
}

func NewTransaction(to uint32, value uint32, data []string) *Transaction {
  tx := Transaction{sender: "1234567890", recipient: to, value: value}
  tx.fee = 0//uint32((ContractFee + MemoryFee * float32(len(tx.data))) * 1e8)

  // Serialize the data
  tx.data = make([]string, len(data))
  for i, val := range data {
    instr, err := CompileInstr(val)
    if err != nil {
      fmt.Printf("compile error:%d %v", i+1, err)
    }

    tx.data[i] = instr
  }

  b:= []byte(tx.MarshalRlp())
  hash := sha256.Sum256(b)
  tx.addr = hex.EncodeToString(hash[0:19])

  return &tx
}

func (tx *Transaction) MarshalRlp() []byte {
  // Prepare the transaction for serialization
  preEnc := []interface{}{
    "0", // TODO last Tx
    tx.sender,
    // XXX In the future there's no need to cast to string because they'll end up being big numbers (strings)
    Uitoa(tx.recipient),
    Uitoa(tx.value),
    Uitoa(tx.fee),
    tx.data,
  }

  return []byte(RlpEncode(preEnc))
}

func InitFees() {
  // Base for 2**60
  b60 := new(big.Int)
  b60.Exp(big.NewInt(2), big.NewInt(64), big.NewInt(0))
  // Base for 2**80
  b80 := new(big.Int)
  b80.Exp(big.NewInt(2), big.NewInt(80), big.NewInt(0))

  StepFee.Div(b60, big.NewInt(64))
  //fmt.Println("StepFee:", StepFee)

  TxFee.Exp(big.NewInt(2), big.NewInt(64), big.NewInt(0))
  //fmt.Println("TxFee:", TxFee)

  ContractFee.Exp(big.NewInt(2), big.NewInt(64), big.NewInt(0))
  //fmt.Println("ContractFee:", ContractFee)

  MemFee.Div(b60, big.NewInt(4))
  //fmt.Println("MemFee:", MemFee)

  DataFee.Div(b60, big.NewInt(16))
  //fmt.Println("DataFee:", DataFee)

  CryptoFee.Div(b60, big.NewInt(16))
  //fmt.Println("CrytoFee:", CryptoFee)

  ExtroFee.Div(b60, big.NewInt(16))
  //fmt.Println("ExtroFee:", ExtroFee)

  Period1Reward.Mul(b80, big.NewInt(1024))
  //fmt.Println("Period1Reward:", Period1Reward)

  Period2Reward.Mul(b80, big.NewInt(512))
  //fmt.Println("Period2Reward:", Period2Reward)

  Period3Reward.Mul(b80, big.NewInt(256))
  //fmt.Println("Period3Reward:", Period3Reward)

  Period4Reward.Mul(b80, big.NewInt(128))
  //fmt.Println("Period4Reward:", Period4Reward)
}
