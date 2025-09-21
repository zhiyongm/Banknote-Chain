// storage is a key-value database and its interfaces indeed
// the information of block will be saved in storage

package storage

import (
	"blockcooker/core"
	"errors"
	"github.com/ethereum/go-ethereum/ethdb"
	"log"
)

var (
	BLK_HEADER_PFX = []byte("H")

	BLK_PFX = []byte("K")
)

type Storage struct {
	//dbFilePath string // path to the database

	DB ethdb.Database
}

// new a storage, build a bolt datase
func NewStorage(statedb ethdb.Database) *Storage {
	//BlockDBPath := path.Join(cc.StoragePath, "blockdb", strconv.FormatUint(cc.NodeID, 10))
	//errMkdir := os.MkdirAll(BlockDBPath, os.ModePerm)
	//fmt.Println(BlockDBPath)
	//if errMkdir != nil {
	//	log.Panic(errMkdir)
	//}
	//
	s := &Storage{}

	//db, err := database.New(BlockDBPath, cc.CacheSize, 128, "", false, false)
	//if err != nil {
	//	log.Panic(err)
	//}

	s.DB = statedb

	return s
}

// update the newest block in the database
// 存储最新区块的hash，便于程序重启后还能接着出区块
func (s *Storage) UpdateNewestBlock(newestbhash []byte) {

	err := s.DB.Put([]byte("OnlyNewestBlock"), newestbhash)
	if err != nil {
		log.Panic()
	}

}

// add a blockheader into the database
// 这个函数只被AddBlock内部调用，所以不用管
func (s *Storage) AddBlockHeader(blockhash []byte, bh *core.BlockHeader) {

	err := s.DB.Put(append(BLK_HEADER_PFX, blockhash...), bh.Encode())

	if err != nil {
		log.Panic()
	}
}

// add a block into the database
func (s *Storage) StoreBlock(b *core.Block) {

	go func() {
		savedBlockEncoded := b.Encode()
		//sum64 := xxhash.Sum64(savedBlockEncoded)
		//hashBytes := make([]byte, 8)
		//binary.BigEndian.PutUint64(hashBytes, sum64)
		//b.Hash = hashBytes
		err := s.DB.Put(append(BLK_PFX, b.Hash...), savedBlockEncoded)
		if err != nil {
			log.Panic()
		}
	}()

	s.AddBlockHeader(b.Hash, b.Header)
	s.UpdateNewestBlock(b.Hash)

}

func (s *Storage) StoreBlockSync(b *core.Block) {

	savedBlockEncoded := b.Encode()
	//sum64 := xxhash.Sum64(savedBlockEncoded)
	//hashBytes := make([]byte, 8)
	//binary.BigEndian.PutUint64(hashBytes, sum64)
	//b.Hash = hashBytes
	err := s.DB.Put(append(BLK_PFX, b.Hash...), savedBlockEncoded)
	if err != nil {
		log.Panic()
	}

	s.AddBlockHeader(b.Hash, b.Header)
	s.UpdateNewestBlock(b.Hash)

}

// read a blockheader from the database
// 从数据库中读取区块头，区块头和区块是分开的，区块头存储在状态数据库中，区块单独存储在blotdb中
func (s *Storage) GetBlockHeader(bhash []byte) (*core.BlockHeader, error) {
	var res *core.BlockHeader

	bh_encoded, err := s.DB.Get(append(BLK_HEADER_PFX, bhash...))
	if err != nil {
		return nil, errors.New("the block is not existed")
	}
	res = core.DecodeBH(bh_encoded)

	return res, err
}

// read a block from the database
// 从数据库中读取区块，区块头和区块是分开的，区块头存储在状态数据库中，区块单独存储在blotdb中
func (s *Storage) GetBlock(bhash []byte) (*core.Block, error) {
	var res *core.Block

	b_encoded, err := s.DB.Get(append(BLK_PFX, bhash...))
	if err != nil {
		return nil, errors.New("the block is not existed")
	}
	res = core.DecodeB(b_encoded)

	return res, err
}

// read the Newest block hash
// 从状态数据库中读取最新区块的hash
func (s *Storage) GetNewestBlockHash() ([]byte, error) {
	var nhb []byte
	//err := s.BlockDB.View(func(tx *bolt.Tx) error {
	// the bucket has the only key "OnlyNewestBlock"
	//nhb, err := s.BlockDB.Get([]byte("OnlyNewestBlock"))
	nhb, err := s.DB.Get([]byte("OnlyNewestBlock"))

	if nhb == nil {
		return nil, errors.New("cannot find the newest block hash")
	}

	return nhb, err
}
