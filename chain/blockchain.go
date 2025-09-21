// Here the blockchain structrue is defined
// each node in this system will maintain a blockchain object.

package chain

import (
	"blockcooker/banknoteChain"
	"blockcooker/core"
	"blockcooker/resultWriter"
	"blockcooker/storage"
	"blockcooker/txGenerator"
	"blockcooker/txPool"
	"blockcooker/utils"
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	log "github.com/sirupsen/logrus"
)

type BlockChain struct {
	GoroutineNum int
	db           ethdb.Database // the leveldb database to store in the disk, for status trie 存储状态的lvldb
	triedb       *trie.Database // the trie database which helps to store the status trie
	ChainConfig  *utils.Config  // the chain configuration, which can help to identify the chain

	CurrentBlock *core.Block      // the top block in this blockchain
	Storage      *storage.Storage // Storage is the bolt-db to store the blocks
	TxPool       *txPool.TxPool

	SignSimulator    *utils.SignSimulator
	StartBlockNumber uint64
	ProcessedTxCount uint64
	resultWriter     *resultWriter.ResultWriter
	ToBeAddedBlocks  chan *core.Block

	BanknoteState *banknoteChain.BanknoteChainState

	USDTDatasetReader          *txGenerator.USDTDatasetTxGeneratorFromCSV
	BanknoteDatasetBatchReader *txGenerator.BanknoteDatasetBatchReader
	BanknoteRandomReader       *txGenerator.BanknoteRandomReader
}

// Get the transaction root, this root can be used to check the transactions
// 获取交易树根
func GetTxTreeRoot(txs []*core.Transaction) []byte {
	// use a memory trie database to do this, instead of disk database
	triedb := trie.NewDatabase(rawdb.NewMemoryDatabase())
	transactionTree := trie.NewEmpty(triedb)
	for _, tx := range txs {
		transactionTree.Update(tx.TxHash.Bytes(), tx.Encode())
	}
	return transactionTree.Hash().Bytes()
}

// generate (mine) a block, this function return a block
// 产生一个区块，并没有将这个区块添加到区块链上面,leader节点会调用
func (bc *BlockChain) GenerateBlockForLeader(txs []*core.Transaction, BanknoteTxs []*banknoteChain.BanknoteTx, blockNumber uint64) *core.Block {
	time_start := time.Now()

	fixedTime := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)

	bh := &core.BlockHeader{
		ParentBlockHash: bc.CurrentBlock.Hash,
		Number:          blockNumber,
		Time:            fixedTime, // 固定时间，为了保证状态根多次运行的一致性
	}
	rt, modify_account_count := bc.ProcessTxs(txs)
	modify_account_count = modify_account_count
	bh.StateRoot = rt.Bytes()

	bh.TxRoot = GetTxTreeRoot(txs)

	bh.Miner = bc.ChainConfig.NodeID

	b := core.NewBlock(bh, txs)
	b.Hash = b.Header.Hash()

	b.BanknoteBlock = ProcessBanknoteTxs(BanknoteTxs, bc.BanknoteState, bc.ChainConfig, blockNumber)

	time_duration := time.Since(time_start)
	bc.resultWriter.CsvEntityChan <- &resultWriter.ResultEntity{
		PublicIP:    bc.ChainConfig.PublicIP,
		Mode:        bc.ChainConfig.ExpMode,
		BlockNumber: b.Header.Number,
		BlockHash:   hex.EncodeToString(b.Hash),
		TxNumber:    uint64(len(b.Body)) + b.BanknoteBlock.BanknoteTxCount,
		ActionType:  "BlkGenerator",
		ProcessTime: int64(time_duration.Milliseconds()),
		TPS:         float64(bc.ChainConfig.TxCountPerBlock) / float64(time_duration.Microseconds()) * 1000 * 1000,
	}
	fmt.Println("区块处理时间", time_duration.Milliseconds(), "ms", "TPS", float64(bc.ChainConfig.TxCountPerBlock)/float64(time_duration.Milliseconds())*1000.0)
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
	b := core.NewBlock(bh, body)

	bc.BanknoteState = banknoteChain.NewBanknoteChainStateGenisis(
		bc.ChainConfig, 0)

	b.BanknoteBlock = banknoteChain.NewGenisisBanknoteBlock(
		bc.ChainConfig.BanknoteChainPLedgerNumber, bc.BanknoteState.PLedgerStateRoots)

	b.Hash = b.Header.Hash()
	return b
}

func (bc *BlockChain) AddGenisisBlock(gb *core.Block) {
	bc.Storage.StoreBlockSync(gb)
	newestHash, err := bc.Storage.GetNewestBlockHash()
	if err != nil {
		log.Panic()
	}
	curb, err := bc.Storage.GetBlock(newestHash)
	if err != nil {
		log.Panic()
	}
	bc.CurrentBlock = curb
	fmt.Println(bc.PrintBlockChain())

}

func (bc *BlockChain) AddBlock(b *core.Block) {
	time_now := time.Now()

	if b.Header.Number != bc.CurrentBlock.Header.Number+1 {
		fmt.Println("the block height is not correct")
		log.Info(b.Header.Number, " ", bc.CurrentBlock.Header.Number, " err block number")
	}

	if !bytes.Equal(b.Header.ParentBlockHash, bc.CurrentBlock.Hash) {
		fmt.Println("err parent block hash")

	}

	if bc.ChainConfig.NodeMode == "leader" {
		log.Info("the world state is already existed, no need to process the transactions again!!!")

		bc.Storage.StoreBlock(b)
		bc.CurrentBlock = b
	} else {

		bc.ProcessTxs(b.Body)
		var banknote_txs []*banknoteChain.BanknoteTx
		for _, bntx := range b.BanknoteBlock.C_ledger {
			banknote_txs = append(banknote_txs, bntx)
		}
		for _, bntx := range b.BanknoteBlock.P_ledgers {
			banknote_txs = append(banknote_txs, bntx...)
		}
		ProcessBanknoteTxs(banknote_txs, bc.BanknoteState, bc.ChainConfig, b.Header.Number)
		bc.Storage.StoreBlock(b)
		bc.CurrentBlock = b
	}

	time_duration := time.Since(time_now)

	tps := float64(bc.ChainConfig.TxCountPerBlock) / float64(time_duration.Microseconds()) * 1000 * 1000
	if bc.ChainConfig.NodeMode == "leader" {
		tps = 0
	}
	bc.resultWriter.CsvEntityChan <- &resultWriter.ResultEntity{
		PublicIP:    bc.ChainConfig.PublicIP,
		Mode:        bc.ChainConfig.ExpMode,
		BlockNumber: b.Header.Number,
		BlockHash:   hex.EncodeToString(b.Hash),
		TxNumber:    uint64(len(b.Body)) + b.BanknoteBlock.BanknoteTxCount,
		ActionType:  "BlkAdder",
		DelayTime:   b.Header.MessageDelayTime.Milliseconds(),
		ProcessTime: int64(time_duration.Milliseconds()),
		TPS:         tps,
	}

}

func (bc *BlockChain) ProcessTxs(txs []*core.Transaction) (common.Hash, int) {
	modified_account_map := make(map[common.Address]bool)
	if len(txs) == 0 {
		return common.BytesToHash(bc.CurrentBlock.Header.StateRoot), 0
	}
	st, err := trie.New(trie.TrieID(common.BytesToHash(bc.CurrentBlock.Header.StateRoot)), bc.triedb)
	if err != nil {
		log.Panic(err)
	}
	cnt := 0
	for _, tx := range txs {
		atomic.AddUint64(&bc.ProcessedTxCount, 1)
		bc.SignSimulator.SimulateVerify() // 模拟验签
		s_state_enc, _ := st.Get([]byte(tx.Sender.Bytes()))
		var s_state *core.AccountState
		if s_state_enc == nil {
			ib := new(big.Int)
			ib.Add(ib, bc.ChainConfig.InitBalance)
			s_state = &core.AccountState{
				Nonce:   0,
				Balance: ib,
			}
		} else {
			s_state = core.DecodeAS(s_state_enc)
		}
		modified_account_map[tx.Sender] = true
		if s_state.Balance == nil {
			log.Info("Error: nil balance in account state of account ", tx.Sender.String())
		}
		s_state.Deduct(tx.Value)

		cnt++
		s_state.Nonce++
		st.Update(tx.Sender.Bytes(), s_state.Encode())

		r_state_enc, _ := st.Get(tx.Recipient.Bytes())
		var r_state *core.AccountState
		if r_state_enc == nil {
			ib := new(big.Int)
			ib.Add(ib, bc.ChainConfig.InitBalance)
			r_state = &core.AccountState{
				Nonce:   0,
				Balance: ib,
			}
		} else {
			r_state = core.DecodeAS(r_state_enc)
		}

		if s_state.Balance == nil {
			log.Info("Error: nil balance in account state of account ", tx.Recipient.String())
		}

		r_state.Deposit(tx.Value)
		r_state.Nonce++
		modified_account_map[tx.Recipient] = true
		st.Update(tx.Recipient.Bytes(), r_state.Encode())
		cnt++
	}

	if cnt == 0 {
		return common.BytesToHash(bc.CurrentBlock.Header.StateRoot), 0
	}
	rt, ns := st.Commit(false)
	if ns != nil {
		err = bc.triedb.Update(trie.NewWithNodeSet(ns))
		if err != nil {
			log.Panic()
		}
		err = bc.triedb.Commit(rt, false)
		if err != nil {
			log.Panic(err)
		}
	}

	return rt, len(modified_account_map)
}

// new a blockchain.
func NewBlockChain(cc *utils.Config) (*BlockChain, error) {

	lvldbPath := filepath.Join(cc.StoragePath, "BlockDB", strconv.FormatUint(cc.NodeID, 10))
	db, _ := rawdb.NewLevelDBDatabase(lvldbPath, cc.CacheSize, 32, "BlockDB", false)

	rwter := resultWriter.NewResultWriter(cc.OutputDatasetPath)

	fmt.Println("Generating a new blockchain", db)
	bc := &BlockChain{
		GoroutineNum:    runtime.NumCPU(),
		db:              db,
		Storage:         storage.NewStorage(db),
		ChainConfig:     cc,
		ToBeAddedBlocks: make(chan *core.Block, 100),
		TxPool:          txPool.NewTxPool(cc.TxPoolSize),
		SignSimulator:   utils.NewRSASimulator(),
		resultWriter:    rwter,
	}

	bc.resultWriter = resultWriter.NewResultWriter(bc.ChainConfig.OutputDatasetPath)

	curHash, err := bc.Storage.GetNewestBlockHash()

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
	currentBlock, err := bc.Storage.GetBlock(curHash)
	if err != nil {
		log.Panic()
	}

	bc.CurrentBlock = currentBlock
	triedb := trie.NewDatabaseWithConfig(db, &trie.Config{
		Cache:     cc.CacheSize,
		Preimages: true,
	})
	bc.triedb = triedb
	_, err = trie.New(trie.TrieID(common.BytesToHash(currentBlock.Header.StateRoot)), triedb)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println("The status trie can be built")

	fmt.Println("Generated a new blockchain successfully")
	return bc, nil
}

// close a blockChain, close the database inferfaces
func (bc *BlockChain) CloseBlockChain() {
	bc.Storage.DB.Close()
	bc.triedb.CommitPreimages()
}

// print the details of a blockchain
func (bc *BlockChain) PrintBlockChain() string {
	res := "=====================\n"
	res += fmt.Sprintf("Current Block Number: %d\n", bc.CurrentBlock.Header.Number)
	res += fmt.Sprintf("Current Block Hash: %s\n", hex.EncodeToString(bc.CurrentBlock.Hash))
	res += fmt.Sprintf("Parent Block Hash: %s\n", hex.EncodeToString(bc.CurrentBlock.Header.ParentBlockHash))
	res += fmt.Sprintf("Current Block State Root: %s\n", hex.EncodeToString(bc.CurrentBlock.Header.StateRoot))
	res += fmt.Sprintf("Current Block Tx Root: %s\n", hex.EncodeToString(bc.CurrentBlock.Header.TxRoot))
	res += fmt.Sprintf("Current Block Time: %s\n", bc.CurrentBlock.Header.Time.String())
	res += fmt.Sprintf("Current Block Miner: %d\n", bc.CurrentBlock.Header.Miner)
	res += fmt.Sprintf("Current Tx Pool Size: %d\n", len(bc.TxPool.TxQueue))
	res += fmt.Sprintf("Current To Be Packaged Blocks Channel Size: %d\n", len(bc.ToBeAddedBlocks))
	res += fmt.Sprintf("Current Block DataSet Tx Count: %d\n", uint64(bc.ChainConfig.TxCountPerBlock))
	res += fmt.Sprintf("Current Block Normal Tx Count: %d\n", len(bc.CurrentBlock.BanknoteBlock.C_ledger))
	res += fmt.Sprintf("Current Block Banknote Tx Count: %d\n", bc.CurrentBlock.BanknoteBlock.BanknoteTxCount)
	res += fmt.Sprintf("To Be Packaged block Count: %d\n", len(bc.ToBeAddedBlocks))

	res += "=====================\n"

	return res
}
