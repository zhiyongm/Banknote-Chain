package storage

import (
	"bytes"
	"coinpurse/core"
	"coinpurse/database"
	"coinpurse/params"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	BLK_HEADER_PFX = []byte("H")

	BLK_PFX = []byte("K")

	BN_PFX       = []byte("B")
	BN_INDEX_PFX = []byte("Z")
)

type Storage struct {
	dbFilePath string

	BlockDB        *database.Database
	StateDB        ethdb.Database
	BNDB           *leveldb.DB
	BNDBMemorylist []*BNDBMemory

	VS_StateDBs []*leveldb.DB
}

type BNDBMemory struct {
	bnMap        map[string]*core.Banknote
	BNDBMemoryLK *sync.RWMutex
}

func NewStorage(cc *params.ChainConfig, statedb ethdb.Database) *Storage {

	dir := cc.BlockDBPath
	errMkdir := os.MkdirAll(dir, os.ModePerm)
	fmt.Println(dir)
	if errMkdir != nil {
		log.Panic(errMkdir)
	}

	s := &Storage{
		dbFilePath: cc.BlockDBPath,
	}

	s.StateDB = statedb

	s.BNDBMemorylist = make([]*BNDBMemory, 0)
	for i := 0; i < int(cc.VSNum); i++ {
		s.BNDBMemorylist = append(s.BNDBMemorylist, &BNDBMemory{
			bnMap:        make(map[string]*core.Banknote),
			BNDBMemoryLK: &sync.RWMutex{},
		})

	}
	return s
}

func (s *Storage) UpdateNewestBlock(newestbhash []byte) {

	err := s.StateDB.Put([]byte("OnlyNewestBlock"), newestbhash)
	if err != nil {
		log.Panic()
	}

}

func (s *Storage) AddBlockHeader(blockhash []byte, bh *core.BlockHeader) {

	err := s.StateDB.Put(append(BLK_HEADER_PFX, blockhash...), bh.Encode())

	if err != nil {
		log.Panic()
	}
}

func (s *Storage) AddBlock(b *core.Block) {

	banknoteshards := b.BanknoteShardBlocks

	wg := sync.WaitGroup{}
	wg.Add(len(banknoteshards))

	bns_hashes := make([][]byte, 0)
	for _, banknoteshard := range banknoteshards {
		go func(banknoteshard *core.BanknoteShardBlock) {
			var buff bytes.Buffer
			enc := gob.NewEncoder(&buff)
			if banknoteshard != nil {
				err := enc.Encode(banknoteshard)
				if err != nil {

					log.Panic(err)
				}
			}

			BlkNumberBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(BlkNumberBytes, b.Header.Number)

			key := append(BLK_PFX, BlkNumberBytes...)
			key = append(key, []byte("~")...)
			key = append(key, []byte(string(banknoteshard.VSNumber))...)

			err := s.StateDB.Put(key, buff.Bytes())
			if err != nil {
				log.Panic()
			}
			wg.Done()
		}(banknoteshard)
	}
	wg.Wait()

	saveblock := &core.SavedBlock{
		Header:              b.Header,
		Body:                b.Body,
		Hash:                nil,
		BlockType:           b.BlockType,
		BNCleanedStateCount: b.BNCleanedStateCount,
		BNsHashes:           bns_hashes,
	}

	savedBlockEncoded := saveblock.Encode()
	sum64 := xxhash.Sum64(savedBlockEncoded)
	hashBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(hashBytes, sum64)
	saveblock.Hash = hashBytes

	err := s.StateDB.Put(append(BLK_PFX, b.Hash...), savedBlockEncoded)
	fmt.Println("存储了！block hash", b.Hash)
	if err != nil {
		log.Panic()
	}
	s.AddBlockHeader(b.Hash, b.Header)
	s.UpdateNewestBlock(b.Hash)

}

func (s *Storage) GetBlockHeader(bhash []byte) (*core.BlockHeader, error) {
	var res *core.BlockHeader

	bh_encoded, err := s.StateDB.Get(append(BLK_HEADER_PFX, bhash...))
	if err != nil {
		return nil, errors.New("the block is not existed")
	}
	res = core.DecodeBH(bh_encoded)

	return res, err
}

func (s *Storage) GetBlock(bhash []byte) (*core.Block, error) {
	var res *core.Block

	b_encoded, err := s.StateDB.Get(append(BLK_PFX, bhash...))
	if err != nil {
		return nil, errors.New("the block is not existed")
	}
	res = core.DecodeB(b_encoded)

	return res, err
}

func (s *Storage) GetNewestBlockHash() ([]byte, error) {
	var nhb []byte

	nhb, err := s.StateDB.Get([]byte("OnlyNewestBlock"))

	if nhb == nil {
		return nil, errors.New("cannot find the newest block hash")
	}

	return nhb, err
}

func (s *Storage) SetBN(bn *core.Banknote, shardNum uint64) {

	bndbmemoryLK := s.BNDBMemorylist[shardNum].BNDBMemoryLK
	BNDBMemory1 := s.BNDBMemorylist[shardNum].bnMap
	bndbmemoryLK.Lock()
	BNDBMemory1[bn.BNID] = bn
	bndbmemoryLK.Unlock()

}

func (s *Storage) DeleteBN(bn *core.Banknote) {

	bndbmemoryLK := s.BNDBMemorylist[bn.VShardNumber].BNDBMemoryLK
	BNDBMemory1 := s.BNDBMemorylist[bn.VShardNumber].bnMap
	bndbmemoryLK.Lock()
	delete(BNDBMemory1, bn.BNID)
	bndbmemoryLK.Unlock()
}

func (s *Storage) SetBNMemory(bn *core.Banknote, shardNum uint64) {

	bndbmemoryLK := s.BNDBMemorylist[shardNum].BNDBMemoryLK
	BNDBMemory1 := s.BNDBMemorylist[shardNum].bnMap
	bndbmemoryLK.Lock()
	BNDBMemory1[bn.BNID] = bn

	bndbmemoryLK.Unlock()

}

func (s *Storage) GetBNbyHashMemory(bnHash string, shardNum uint64) (*core.Banknote, error) {
	bndbmemoryLK := s.BNDBMemorylist[shardNum].BNDBMemoryLK
	BNDBMemory1 := s.BNDBMemorylist[shardNum].bnMap

	bndbmemoryLK.Lock()
	bn, ok := BNDBMemory1[bnHash]
	bndbmemoryLK.Unlock()
	if !ok {
		return nil, errors.New("cannot find the banknote")
	}
	return bn, nil

}
func (s *Storage) DeleteBNMemory(bn *core.Banknote, shardNum uint64) {

	bndbmemoryLK := s.BNDBMemorylist[shardNum].BNDBMemoryLK
	BNDBMemory1 := s.BNDBMemorylist[shardNum].bnMap
	bndbmemoryLK.Lock()

	delete(BNDBMemory1, bn.BNID)
	bndbmemoryLK.Unlock()
}
