package core

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
)

type BNTxRecord struct {
	Sender       common.Address
	Recipient    common.Address
	Amount       *big.Int
	BN_tx_type   TxType
	BanknoteHash string
	BnValue      *big.Int
	ShardId      int
	BnNew1Hash   string
	BnNew1Value  *big.Int
	BnNew2Hash   string
	BnNew2Value  *big.Int
}

func (tx *BNTxRecord) Encode() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

type BNTxGroup []*BNTxRecord

type TxsInABlock struct {
	NormalTxs  []*Transaction
	BNTxGroups []BNTxGroup
}

type BanknoteShardBlock struct {
	VSNumber  uint64
	Banknotes []*Banknote
}

type Banknote struct {
	BNID string

	VShardNumber        uint64
	LastSeenBlockNumber uint64

	Denomination *big.Int
	Owner        common.Address

	Nonce uint64

	Sig []byte
}

func (bn *Banknote) Encode() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(bn)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

func DecodeBN(b []byte) *Banknote {
	var bn Banknote
	decoder := gob.NewDecoder(bytes.NewReader(b))
	err := decoder.Decode(&bn)
	if err != nil {
		log.Panic(err)
	}
	return &bn
}

func (bn *Banknote) hash() common.Hash {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(bn.Nonce))
	h := crypto.Keccak256Hash(bn.Encode())
	return h
}

func (bn *Banknote) TransferOwner(newOwner common.Address) {
	bn.Owner = newOwner

}

func (bn *Banknote) CheckSplit(newBNs []Banknote) bool {
	var totalValue = new(big.Int)
	for _, newBN := range newBNs {
		totalValue.Add(totalValue, newBN.Denomination)
	}

	if totalValue.Cmp(bn.Denomination) != 0 {
		return false
	}

	return true
}

func NewBanknoteWithHash(hashstring string, vsNum uint64, activeBlockNum uint64, owner common.Address, producer common.Address, nonce uint64, denomination *big.Int) *Banknote {
	bn := &Banknote{
		VShardNumber:        vsNum,
		LastSeenBlockNumber: activeBlockNum,
		Owner:               owner,

		Nonce:        nonce,
		Denomination: denomination,
	}

	bn.BNID = hashstring

	return bn
}
