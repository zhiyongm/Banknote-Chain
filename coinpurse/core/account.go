package core

import (
	"bytes"
	"coinpurse/utils"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type AccountState struct {
	AcAddress      utils.Address
	Nonce          uint64
	Balance        *big.Int
	StorageRoot    []byte
	CodeHash       []byte
	BanknoteHashes []common.Hash
}

func (as *AccountState) Deduct(val *big.Int) bool {
	if as.Balance.Cmp(val) <= 0 {
		return false
	}
	as.Balance.Sub(as.Balance, val)
	return true
}

func (s *AccountState) Deposit(value *big.Int) {
	s.Balance.Add(s.Balance, value)
}

func (as *AccountState) Encode() []byte {
	var buff bytes.Buffer
	encoder := gob.NewEncoder(&buff)
	err := encoder.Encode(as)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

func DecodeAS(b []byte) *AccountState {
	var as AccountState

	decoder := gob.NewDecoder(bytes.NewReader(b))
	err := decoder.Decode(&as)
	if err != nil {
		log.Panic(err)
	}
	return &as
}

func (as *AccountState) Hash() []byte {
	h := sha256.Sum256(as.Encode())
	return h[:]
}
