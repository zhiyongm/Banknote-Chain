package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type TxType int

const (
	NewBN TxType = iota
	BNTransfer
	BNRecycle
	BNSplit
	NewBN_State
	BNTransfer_State
	BNRecycle_State
	NormalTx
)

type MultiTxs struct {
	NormalTxs  []*Transaction
	BnblockTxs []*BNTxRecord
	BnStateTxs []*BNTxRecord
}

type Transaction struct {
	Sender    common.Address
	Recipient common.Address
	Nonce     uint64
	Value     *big.Int
	TxHash    common.Hash

	TxType   TxType
	VSNum    uint64
	BNID     string
	BNnew1ID string
	BNnew2ID string
	Sig      []byte
}

func (tx *Transaction) PrintTx() string {

	vals := fmt.Sprintf("Sender: %v\nRecipient: %v\nValue: %v\nTxHash: %v\nTime: %v\n",
		tx.Sender.String(),
		tx.Recipient.String(),
		tx.Value.String(),
		tx.TxHash.String())

	return vals
}

func (tx *Transaction) Encode() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func DecodeTx(to_decode []byte) *Transaction {
	var tx Transaction

	decoder := gob.NewDecoder(bytes.NewReader(to_decode))
	err := decoder.Decode(&tx)
	if err != nil {
		log.Panic(err)
	}

	return &tx
}

func NewTransaction(sender, recipient common.Address, value *big.Int, nonce uint64) *Transaction {
	tx := &Transaction{
		Sender:    sender,
		Recipient: recipient,
		Value:     value,
		Nonce:     nonce,
		TxType:    NormalTx,
	}

	hash := sha256.Sum256(tx.Encode())

	tx.TxHash = common.BytesToHash(hash[:])
	return tx
}
func NewBNTransaction(sender, recipient common.Address, value *big.Int, nonce uint64, TxType TxType, VSid uint64, BNHash string) *Transaction {
	tx := &Transaction{
		Sender:    sender,
		Recipient: recipient,
		Value:     value,
		Nonce:     nonce,
		TxType:    TxType,
		VSNum:     VSid,
		BNID:      BNHash,
	}

	hash := sha256.Sum256(tx.Encode())

	tx.TxHash = common.BytesToHash(hash[:])
	return tx
}
