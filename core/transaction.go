// Definition of transaction

package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"log"
	"math/big"
	"time"
)

type TxType int

type MultiTxs struct {
	NormalTxs []*Transaction
}

type Transaction struct {
	Sender    common.Address
	Recipient common.Address
	Nonce     uint64
	Value     *big.Int
	TxHash    common.Hash
	Time      time.Time // TimeStamp the tx proposed.
	Sig       []byte
}

func (tx *Transaction) PrintTx() string {

	vals := fmt.Sprintf("Sender: %v\nRecipient: %v\nValue: %v\nTxHash: %v\n Time: %v\n",
		tx.Sender.String(),
		tx.Recipient.String(),
		tx.Value.String(),
		tx.TxHash.String(),
		tx.Time.String())

	return vals
}

// Encode transaction for storing
func (tx *Transaction) Encode() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

// Decode transaction
func DecodeTx(to_decode []byte) *Transaction {
	var tx Transaction

	decoder := gob.NewDecoder(bytes.NewReader(to_decode))
	err := decoder.Decode(&tx)
	if err != nil {
		log.Panic(err)
	}

	return &tx
}

// new a transaction
func NewTransaction(sender, recipient common.Address, nonce uint64, value *big.Int, txHash common.Hash, time time.Time) *Transaction {
	return &Transaction{
		Sender:    sender,
		Recipient: recipient,
		Nonce:     nonce,
		Value:     value,
		TxHash:    txHash,
		Time:      time,
		Sig:       make([]byte, 0),
	}
}
