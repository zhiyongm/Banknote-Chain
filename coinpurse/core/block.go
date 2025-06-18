package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type BlockType int

const (
	Clear BlockType = iota
	Generate
	Normal
)

type BlockHeader struct {
	ParentBlockHash         []byte
	StateRoot               []byte
	TxRoot                  []byte
	Number                  uint64
	Time                    time.Time
	Miner                   uint64
	NormalTxCount           uint64
	BNCount                 uint64
	BNStateCount            uint64
	TotalTxCount            uint64
	Cleaned_BN_modify_count uint64
}

func (bh *BlockHeader) Encode() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(bh)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

func DecodeBH(b []byte) *BlockHeader {
	var blockHeader BlockHeader

	decoder := gob.NewDecoder(bytes.NewReader(b))
	err := decoder.Decode(&blockHeader)
	if err != nil {
		log.Panic(err)
	}

	return &blockHeader
}

func (bh *BlockHeader) Hash() []byte {
	hash := sha256.Sum256(bh.Encode())
	return hash[:]
}

func (bh *BlockHeader) PrintBlockHeader() string {
	vals := []interface{}{
		hex.EncodeToString(bh.ParentBlockHash),
		hex.EncodeToString(bh.StateRoot),
		hex.EncodeToString(bh.TxRoot),
		bh.Number,
		bh.Time,
	}
	res := fmt.Sprintf("%v\n", vals)
	return res
}

type Block struct {
	Header              *BlockHeader
	Body                []*Transaction
	Hash                []byte
	BanknoteShardBlocks []*BanknoteShardBlock
	BlockType           *BlockType
	BNCleanedStateCount uint64
}

type SavedBlock struct {
	Header              *BlockHeader
	Body                []*Transaction
	Hash                []byte
	BlockType           *BlockType
	BNCleanedStateCount uint64
	BNsHashes           [][]byte
}

func NewBlock(bh *BlockHeader, bb []*Transaction, BSblocks []*BanknoteShardBlock) *Block {

	return &Block{Header: bh, Body: bb, BanknoteShardBlocks: BSblocks}
}
func NewFullBlock(bh *BlockHeader, bb []*Transaction, hashbytes []byte) *Block {
	return &Block{Header: bh, Body: bb, Hash: hashbytes}
}
func (b *Block) PrintBlock() string {

	res := fmt.Sprintf("BlockNumber: %v Hash: %v TotalTxCount: %v NormalTxCount: %v BNCount: %v BNCleanedStateCount: %v\n", b.Header.Number, common.BytesToAddress(b.Hash).String(), b.Header.TotalTxCount, b.Header.NormalTxCount, b.Header.BNCount, b.BNCleanedStateCount)

	return res
}
func (b *Block) PrintBlockDebug() {
	bsbs := b.BanknoteShardBlocks
	fmt.Println("bsbs size", len(bsbs))
	for _, bsb := range bsbs {
		fmt.Println("VSNumber: ", bsb.VSNumber)
		fmt.Println("NumberTxs: ", len(bsb.Banknotes))
		for i, bn := range bsb.Banknotes {
			fmt.Println("Banknote ", i)
			fmt.Println("BNID: ", bn.BNID)
			fmt.Println("VShardNumber: ", bn.VShardNumber)
			fmt.Println("LastSeenBlockNumber: ", bn.LastSeenBlockNumber)
			fmt.Println("Denomination: ", bn.Denomination)
			fmt.Println("Owner: ", bn.Owner)
			fmt.Println("Nonce: ", bn.Nonce)
		}
	}
}

func (b *Block) Encode() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(b)
	if err != nil {

		log.Panic(err)
	}
	return buff.Bytes()
}

func (b *SavedBlock) Encode() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(b)
	if err != nil {

		log.Panic(err)
	}
	return buff.Bytes()
}

func DecodeB(b []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(b))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}

	return &block
}
