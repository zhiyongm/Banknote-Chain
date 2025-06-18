package chain

import (
	"bytes"
	"coinpurse/core"
	"coinpurse/params"
	"coinpurse/storage"
	"coinpurse/utils"
	"fmt"
	"math/big"
	"path"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	log "github.com/sirupsen/logrus"
)

type BlockChain struct {
	SimuFlag     bool
	GoroutineNum int
	db           ethdb.Database
	triedb       *trie.Database
	triedb_VSs   []*trie.Database
	ChainConfig  *params.ChainConfig
	CurrentBlock *core.Block
	Storage      *storage.Storage

	BNStateChans    []chan *core.BNTxRecord
	BNMdifiyChans   []chan *core.CoreModify
	BNEnable        bool
	VSNum           uint64
	toDoBlockNumber uint64
	bnReportChans   chan *core.BanknoteShardBlock
	bnReportList    []*core.BanknoteShardBlock

	StartBlockNumber uint64
	RSAsimulator     *utils.RSASimulator

	NewBN_chans      []chan *core.BNTxRecord
	BNTransfer_chans []chan *core.BNTxRecord
	BNRecycle_chans  []chan *core.BNTxRecord
	BNSplit_chans    []chan *core.BNTxRecord

	BNMdifiyChan_Clean        chan *core.CoreModify
	CurrentBNShardStateHashes []*common.Hash
	ProcessedTxCount          uint64
	BNCleanedStateCount       uint64
	ECDSAsimulator            *utils.ECDSASimulator
	BNStateDBPaths            []string
}

func GetTxTreeRoot(txs []*core.Transaction) []byte {

	triedb := trie.NewDatabase(rawdb.NewMemoryDatabase())
	transactionTree := trie.NewEmpty(triedb)
	for _, tx := range txs {
		transactionTree.Update(tx.TxHash.Bytes(), tx.Encode())
	}
	return transactionTree.Hash().Bytes()
}

func (bc *BlockChain) BNWorker(VSID uint64, wg *sync.WaitGroup) {

	sub_worker_number := int(bc.GoroutineNum / int(bc.VSNum))
	if sub_worker_number <= 0 {
		sub_worker_number = 1
	}

	sub_worker_number = sub_worker_number + 1

	var state_chan = bc.BNStateChans[VSID]

	var modify_chan = bc.BNMdifiyChans[VSID]
	var NewBN_chan = bc.NewBN_chans[VSID]
	var BNTransfer_chan = bc.BNTransfer_chans[VSID]
	var BNRecycle_chan = bc.BNRecycle_chans[VSID]
	var BNSplit_chan = bc.BNSplit_chans[VSID]

	var tr *trie.Trie
	if bc.CurrentBNShardStateHashes[VSID] == nil {
		tr = trie.NewEmpty(bc.triedb_VSs[VSID])
		h := tr.Hash()
		bc.CurrentBNShardStateHashes[VSID] = &h
	} else {

		tr, _ = trie.New(trie.TrieID(*bc.CurrentBNShardStateHashes[VSID]), bc.triedb_VSs[VSID])

	}

	newBNCount := 0
	BNTransferCount := 0
	BNRecycleCount := 0
	BNSplitCount := 0

FOR:
	for {
		select {
		case txstate := <-state_chan:
			{

				switch txstate.BN_tx_type {
				case core.NewBN:
					{
						NewBN_chan <- txstate
						newBNCount++
					}
				case core.BNTransfer:
					{
						BNTransfer_chan <- txstate
						BNTransferCount++
					}
				case core.BNRecycle:
					{
						BNRecycle_chan <- txstate
						BNRecycleCount++
					}

				case core.BNSplit:
					{
						BNSplit_chan <- txstate
						BNSplitCount++
					}

				}
			}
		default:
			{

				break FOR
			}
		}
	}

	subwg := sync.WaitGroup{}
	subwg.Add(sub_worker_number)
	for i := 0; i < sub_worker_number; i++ {
		go func() {
		FOR1:
			for {
				select {
				case txstate := <-NewBN_chan:
					{

						cm := &core.CoreModify{
							Address: txstate.Recipient,
							Balance: txstate.BnValue,
							IsAdd:   true,
						}
						modify_chan <- cm

						cm = &core.CoreModify{
							Address: txstate.Sender,
							Balance: txstate.BnValue,
							IsAdd:   false,
						}
						modify_chan <- cm

						bn := core.NewBanknoteWithHash(txstate.BanknoteHash, VSID, bc.CurrentBlock.Header.Number, txstate.Recipient, txstate.Sender, 0, txstate.BnValue)

						bc.Storage.SetBNMemory(bn, VSID)

					}
				default:
					{

						subwg.Done()
						break FOR1

					}
				}
			}

		}()
	}
	subwg.Wait()

	subwg = sync.WaitGroup{}
	subwg.Add(sub_worker_number)

	for i := 0; i < sub_worker_number; i++ {
		go func() {
		FOR33:
			for {
				select {
				case txstate := <-BNSplit_chan:
					{

						bn, _ := bc.Storage.GetBNbyHashMemory(txstate.BanknoteHash, VSID)

						if bn != nil {
							bc.Storage.DeleteBNMemory(bn, VSID)

						}

						bn1 := core.NewBanknoteWithHash(txstate.BnNew1Hash,
							xxhash.Sum64([]byte(txstate.BnNew1Hash))%bc.VSNum, bc.CurrentBlock.Header.Number,
							txstate.Sender, txstate.Sender, 0, txstate.BnNew1Value)
						bn2 := core.NewBanknoteWithHash(txstate.BnNew2Hash,
							xxhash.Sum64([]byte(txstate.BnNew2Hash))%bc.VSNum,
							bc.CurrentBlock.Header.Number,
							txstate.Recipient, txstate.Sender, 0, txstate.BnNew2Value)
						bc.Storage.SetBN(bn1, bn1.VShardNumber)

						bc.Storage.SetBN(bn2, bn2.VShardNumber)

					}
				default:
					{

						subwg.Done()

						break FOR33

					}
				}
			}
		}()
	}

	subwg.Wait()

	subwg = sync.WaitGroup{}
	subwg.Add(sub_worker_number)
	for i := 0; i < sub_worker_number; i++ {
		go func() {
		FOR2:
			for {
				select {
				case txstate := <-BNTransfer_chan:
					{

						bn, _ := bc.Storage.GetBNbyHashMemory(txstate.BanknoteHash, VSID)

						if bn == nil {
							bn = core.NewBanknoteWithHash(txstate.BanknoteHash, VSID,
								bc.CurrentBlock.Header.Number, txstate.Recipient,
								txstate.Sender, 0, txstate.BnValue)
							if !bc.SimuFlag {
								bc.Storage.SetBN(bn, VSID)
							}

						}
						if !bc.SimuFlag {
							bc.Storage.DeleteBN(bn)
							bn.Owner = txstate.Recipient
							bc.Storage.SetBN(bn, VSID)
						}

					}
				default:
					{

						subwg.Done()

						break FOR2

					}
				}
			}
		}()
	}

	subwg.Wait()

	subwg = sync.WaitGroup{}
	subwg.Add(sub_worker_number)
	for i := 0; i < sub_worker_number; i++ {
		go func() {
		FOR3:
			for {
				select {
				case txstate := <-BNRecycle_chan:
					{
						cm := &core.CoreModify{
							Address: txstate.Recipient,
							Balance: txstate.BnValue,
							IsAdd:   true,
						}
						modify_chan <- cm

						fmt.Println("delete", txstate.BanknoteHash)
						bn, _ := bc.Storage.GetBNbyHashMemory(txstate.BanknoteHash, VSID)

						bc.Storage.DeleteBNMemory(bn, VSID)
						fmt.Println("delete OK", txstate.BanknoteHash)

					}
				default:
					{

						subwg.Done()
						break FOR3

					}
				}
			}

		}()
	}
	subwg.Wait()

	var wg_state_trie = sync.WaitGroup{}

	wg_state_trie.Add(1)
	go func() {
		BNMdifiyMap := make(map[common.Address]int64)
		BNMdifiyChan_Clean := make(chan *core.CoreModify, bc.ChainConfig.BlockSize+100)
	FOR111:
		for {
			select {
			case data := <-modify_chan:
				{

					if data.IsAdd {
						BNMdifiyMap[data.Address] += data.Balance.Int64()
					} else {
						BNMdifiyMap[data.Address] -= data.Balance.Int64()
					}
				}
			default:
				{
					break FOR111
				}
			}
		}

		for k, v := range BNMdifiyMap {

			atomic.AddUint64(&bc.BNCleanedStateCount, 1)

			BNMdifiyChan_Clean <- &core.CoreModify{
				Address: k,
				Balance: big.NewInt(v),
				IsAdd:   v > 0,
			}
		}

		if len(BNMdifiyMap) == 0 {
			wg_state_trie.Done()
			return
		}
		wg1 := sync.WaitGroup{}
		STATE_work_Number := bc.GoroutineNum/int(bc.VSNum) + 1

		wg1.Add(STATE_work_Number)
		var mu sync.Mutex

		for i := 0; i < STATE_work_Number; i++ {
			go func() {
			FOR:
				for {
					select {
					case data := <-BNMdifiyChan_Clean:
						{
							mu.Lock()
							s_state_enc, _ := tr.Get(data.Address.Bytes())
							mu.Unlock()
							var s_state *core.AccountState
							if s_state_enc == nil {
								ib := new(big.Int)
								ib.Add(ib, params.Init_Balance)
								s_state = &core.AccountState{
									Nonce:   0,
									Balance: ib,
								}
							} else {
								s_state = core.DecodeAS(s_state_enc)
							}
							if data.IsAdd {
								s_state.Deposit(data.Balance)
							} else {
								s_state.Deduct(data.Balance)
							}

							mu.Lock()
							tr.Update(data.Address.Bytes(), s_state.Encode())

							mu.Unlock()

						}
					default:
						{

							wg1.Done()
							break FOR
						}
					}
				}

			}()

		}
		wg1.Wait()

		rt, ns := tr.Commit(false)

		mergednodeset := trie.NewWithNodeSet(ns)
		err := bc.triedb_VSs[VSID].Update(mergednodeset)
		if err != nil {
			panic(err)
		}
		err = bc.triedb_VSs[VSID].Commit(rt, false)
		if err != nil {
			log.Panic(err)
		}

		bc.CurrentBNShardStateHashes[VSID] = &rt

		wg_state_trie.Done()
	}()

	wg_state_trie.Wait()

	wg.Done()

}

func (bc *BlockChain) ProcessTxs(txs []*core.Transaction, bnstatetxs []*core.BNTxRecord, blkNumber uint64) (common.Hash, uint64) {

	if len(txs) == 0 && len(bnstatetxs) == 0 {
		fmt.Println("No transaction in this block", bc.CurrentBlock.Header.Number)
		return common.BytesToHash(bc.CurrentBlock.Header.StateRoot), 0
	}

	for _, bntx := range bnstatetxs {

		bc.BNStateChans[bntx.ShardId] <- bntx
	}

	wg := sync.WaitGroup{}
	wg.Add(int(bc.VSNum))
	for i := 0; i < int(bc.VSNum); i++ {
		go bc.BNWorker(uint64(i), &wg)
	}

	wg.Wait()

	st, err := trie.New(trie.TrieID(common.BytesToHash(bc.CurrentBlock.Header.StateRoot)), bc.triedb)

	if err != nil {
		log.Panic(err)
	}

	for i, tx := range txs {
		atomic.AddUint64(&bc.ProcessedTxCount, 1)

		bc.RSAsimulator.SimulateVerify()
		s_state_enc, _ := st.Get([]byte(tx.Sender.Bytes()))
		var s_state *core.AccountState
		if s_state_enc == nil {

			ib := new(big.Int)
			ib.Add(ib, params.Init_Balance)
			s_state = &core.AccountState{
				Nonce:   uint64(i),
				Balance: ib,
			}
		} else {
			s_state = core.DecodeAS(s_state_enc)
		}

		s_state.Deduct(tx.Value)

		st.Update(tx.Sender.Bytes(), s_state.Encode())

		r_state_enc, _ := st.Get(tx.Recipient.Bytes())
		var r_state *core.AccountState
		if r_state_enc == nil {

			ib := new(big.Int)
			ib.Add(ib, params.Init_Balance)
			r_state = &core.AccountState{
				Nonce:   0,
				Balance: ib,
			}
		} else {
			r_state = core.DecodeAS(r_state_enc)
		}
		r_state.Deposit(tx.Value)

		st.Update(tx.Recipient.Bytes(), r_state.Encode())

	}

	var isModify = false
	for i := 0; i < int(bc.VSNum); i++ {
		if bc.CurrentBNShardStateHashes[i].Bytes() != nil {
			isModify = true
			err := st.Update([]byte("VS111"+strconv.Itoa(i)), bc.CurrentBNShardStateHashes[i].Bytes())
			if err != nil {
				log.Panic(err)
			}

		}
	}

	if isModify {
		rt, ns := st.Commit(false)

		mergednodeset := trie.NewWithNodeSet(ns)
		err = bc.triedb.Update(mergednodeset)

		if err != nil {
			panic(err)
		}

		err = bc.triedb.Commit(rt, false)

		if err != nil {
			log.Panic(err)
		}
		return st.Hash(), 0
	}

	return st.Hash(), 0
}

func (bc *BlockChain) GenerateBlockwithTxs(MultiTxs *core.TxsInABlock) *core.Block {

	var normalTxs []*core.Transaction = MultiTxs.NormalTxs

	var bnblockTxs []*core.BNTxRecord
	var bnstateTxs []*core.BNTxRecord

	for _, bnTxGroup := range MultiTxs.BNTxGroups {
		bnblockTxs = append(bnblockTxs, bnTxGroup[0])
		bnstateTxs = append(bnstateTxs, bnTxGroup[1:]...)
	}

	bh := &core.BlockHeader{
		ParentBlockHash: bc.CurrentBlock.Hash,
		Number:          bc.CurrentBlock.Header.Number + 1,
		Time:            time.Now(),
		TotalTxCount:    uint64(len(normalTxs) + len(bnblockTxs)),
		NormalTxCount:   uint64(len(normalTxs)),
		BNCount:         uint64(len(bnblockTxs)),
		BNStateCount:    uint64(len(bnstateTxs)),
	}

	var BSblocks = make([]*core.BanknoteShardBlock, bc.VSNum)

	for i := 0; i < int(bc.VSNum); i++ {
		BSblocks[i] = &core.BanknoteShardBlock{
			VSNumber:  uint64(i),
			Banknotes: make([]*core.Banknote, 0),
		}
	}

	wgBNBlockTxs := sync.WaitGroup{}
	wgBNBlockTxs.Add(bc.GoroutineNum)

	bnBlockTxschan := make(chan *core.BNTxRecord, bc.ChainConfig.BlockSize+100)
	for _, bn_tx := range bnblockTxs {
		bnBlockTxschan <- bn_tx

	}
	bnblockTxs = nil

	slice_mutex := sync.Mutex{}
	for i := 0; i < bc.GoroutineNum; i++ {
		go func() {

		FFF:
			for {
				select {
				case bn_tx := <-bnBlockTxschan:
					{
						atomic.AddUint64(&bc.ProcessedTxCount, 1)

						bc.RSAsimulator.SimulateVerify()
						bn := core.NewBanknoteWithHash(bn_tx.BanknoteHash, uint64(bn_tx.ShardId), bh.Number, bn_tx.Sender, bn_tx.Recipient, 0, bn_tx.BnValue)

						bn.Sig = make([]byte, 65)
						shard_num := bn_tx.ShardId
						slice_mutex.Lock()
						BSblocks[shard_num].Banknotes = append(BSblocks[shard_num].Banknotes, bn)
						slice_mutex.Unlock()

					}
				default:
					{
						break FFF

					}
				}
			}
			wgBNBlockTxs.Done()
		}()
	}

	for i := 0; i < int(bc.VSNum); i++ {

		BSblocks[i].Banknotes = make([]*core.Banknote, 0)

	}

	rt, cleaned_BN_modify_count := bc.ProcessTxs(normalTxs, bnstateTxs, bh.Number)
	bh.Cleaned_BN_modify_count = cleaned_BN_modify_count
	bh.StateRoot = rt.Bytes()
	bh.TxRoot = GetTxTreeRoot(normalTxs)
	wgBNBlockTxs.Wait()

	b := core.NewBlock(bh, normalTxs, BSblocks)
	b.Header.Miner = 0
	b.Hash = b.Header.Hash()
	b.BNCleanedStateCount = bc.BNCleanedStateCount
	bc.BNCleanedStateCount = 0

	return b
}

func (bc *BlockChain) NewGenisisBlock() *core.Block {
	body := make([]*core.Transaction, 0)

	bh := &core.BlockHeader{
		Number: 0,
	}

	triedb := trie.NewDatabaseWithConfig(bc.db, &trie.Config{
		Cache:     bc.ChainConfig.CacheSize,
		Preimages: true,
	})
	bc.triedb = triedb
	statusTrie := trie.NewEmpty(triedb)
	bh.StateRoot = statusTrie.Hash().Bytes()
	bh.TxRoot = GetTxTreeRoot(body)

	b := core.NewBlock(bh, body, make([]*core.BanknoteShardBlock, 0))
	b.Hash = b.Header.Hash()
	return b
}

func (bc *BlockChain) AddGenisisBlock(gb *core.Block) {
	bc.Storage.AddBlock(gb)
	newestHash, err := bc.Storage.GetNewestBlockHash()
	if err != nil {
		log.Panic()
	}
	curb, err := bc.Storage.GetBlock(newestHash)
	if err != nil {
		log.Panic()
	}
	bc.CurrentBlock = curb
}

func (bc *BlockChain) AddBlock(b *core.Block) {
	if b.Header.Number != bc.CurrentBlock.Header.Number+1 {
		fmt.Println("the block height is not correct")
		return
	}

	if !bytes.Equal(b.Header.ParentBlockHash, bc.CurrentBlock.Hash) {
		fmt.Println("err parent block hash")
		return
	}

	bc.CurrentBlock = b
	bc.Storage.AddBlock(b)
}

func NewBlockChain(cc *params.ChainConfig, db ethdb.Database) (*BlockChain, error) {

	fmt.Println("Generating a new blockchain", db)

	NewBN_chans := make([]chan *core.BNTxRecord, 0)
	BNTransfer_chans := make([]chan *core.BNTxRecord, 0)
	BNRecycle_chans := make([]chan *core.BNTxRecord, 0)
	BNSplitChans := make([]chan *core.BNTxRecord, 0)

	BNMdifiyChans := make([]chan *core.CoreModify, 0)

	triedb_VSs := make([]*trie.Database, 0)
	var bnStateDBpaths []string

	for i := 0; i < int(cc.VSNum); i++ {
		BNMdifiyChans = append(BNMdifiyChans, make(chan *core.CoreModify, (cc.BlockSize+100)*int(cc.VSNum)*2))
		NewBN_chans = append(NewBN_chans, make(chan *core.BNTxRecord, (cc.BlockSize+100)*int(cc.VSNum)*2))
		BNTransfer_chans = append(BNTransfer_chans, make(chan *core.BNTxRecord, (cc.BlockSize+100)*int(cc.VSNum)*2))
		BNRecycle_chans = append(BNRecycle_chans, make(chan *core.BNTxRecord, (cc.BlockSize+100)*int(cc.VSNum)*2))
		BNSplitChans = append(BNRecycle_chans, make(chan *core.BNTxRecord, (cc.BlockSize+100)*int(cc.VSNum)*2))

		var nodeid = utils.RandomString(5)
		pth := path.Join(cc.DBrootPath, "/VSstatedb_"+nodeid)
		bnStateDBpaths = append(bnStateDBpaths, pth)

		var VSstateDB, _ = rawdb.NewLevelDBDatabase(pth, cc.CacheSize, 32, "", false)

		triedbVS := trie.NewDatabaseWithConfig(VSstateDB, &trie.Config{
			Cache:     cc.CacheSize,
			Preimages: true,
		})
		triedb_VSs = append(triedb_VSs, triedbVS)
	}

	bc := &BlockChain{

		BNMdifiyChans: BNMdifiyChans,
		GoroutineNum:  runtime.NumCPU(),
		db:            db,
		ChainConfig:   cc,

		Storage: storage.NewStorage(cc, db),

		VSNum:         cc.VSNum,
		BNEnable:      cc.BNEnable,
		bnReportChans: make(chan *core.BanknoteShardBlock, cc.VSNum),

		RSAsimulator:              utils.NewRSASimulator(),
		ECDSAsimulator:            utils.NewECDSASimulator(),
		NewBN_chans:               NewBN_chans,
		BNRecycle_chans:           BNRecycle_chans,
		BNTransfer_chans:          BNTransfer_chans,
		BNSplit_chans:             BNSplitChans,
		BNMdifiyChan_Clean:        make(chan *core.CoreModify, cc.BlockSize+100),
		CurrentBNShardStateHashes: make([]*common.Hash, cc.VSNum),
		triedb_VSs:                triedb_VSs,
		BNStateDBPaths:            bnStateDBpaths,
	}

	curHash, err := bc.Storage.GetNewestBlockHash()

	bc.BNStateChans = make([]chan *core.BNTxRecord, cc.VSNum)
	bc.bnReportList = make([]*core.BanknoteShardBlock, 0)

	for i := 0; i < int(cc.VSNum); i++ {
		bc.BNStateChans[i] = make(chan *core.BNTxRecord, (cc.BlockSize+100)*int(cc.VSNum)*2)
	}

	if err != nil {
		fmt.Println("There is no existed blockchain in the database. ")

		if err.Error() == "cannot find the newest block hash" {
			genisisBlock := bc.NewGenisisBlock()
			bc.AddGenisisBlock(genisisBlock)
			fmt.Println("New genisis block")
			return bc, nil
		}
		log.Panic(err)
	}

	fmt.Println("Existing blockchain found")
	curb, err := bc.Storage.GetBlock(curHash)
	if err != nil {
		log.Panic()
	}

	bc.CurrentBlock = curb
	triedb := trie.NewDatabaseWithConfig(db, &trie.Config{
		Cache:     cc.CacheSize,
		Preimages: true,
	})
	bc.triedb = triedb

	_, err = trie.New(trie.TrieID(common.BytesToHash(curb.Header.StateRoot)), triedb)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println("The status trie can be built")

	fmt.Println("Generated a new blockchain successfully")
	return bc, nil
}

func (bc *BlockChain) CloseBlockChain() {
	bc.Storage.BlockDB.Close()
	bc.triedb.CommitPreimages()
}

func (bc *BlockChain) PrintBlockChain() string {
	vals := []interface{}{
		bc.CurrentBlock.Header.Number,
		bc.CurrentBlock.Hash,
		bc.CurrentBlock.Header.StateRoot,
		bc.CurrentBlock.Header.Time,
		bc.triedb,
	}
	res := fmt.Sprintf("%v\n", vals)
	fmt.Println(res)
	return res
}
