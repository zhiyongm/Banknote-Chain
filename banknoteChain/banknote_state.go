package banknoteChain

import (
	"blockcooker/utils"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	log "github.com/sirupsen/logrus"
)

type BanknoteChainState struct {
	CurrentBlockNumber  uint64
	CLedgerStateRoot    common.Hash
	PLedgerStateRoots   []common.Hash
	Ethdbs              []ethdb.Database
	Triedbs             []*trie.Database
	ActiveBanknotesPool *ActiveBanknotePool
	SignSimulator       *utils.SignSimulator
}

// TODO 这个是在程序的暂停后重新启动用的，我们的实验中不需要这个功能
func NewBanknoteChainState(cc *utils.Config, currentBlockNumber uint64, ledgerStateRoots []common.Hash) *BanknoteChainState {
	BanknoteChainPLedgerNumber := cc.BanknoteChainPLedgerNumber
	bcs := &BanknoteChainState{
		CurrentBlockNumber: currentBlockNumber,
		PLedgerStateRoots:  ledgerStateRoots,
	}
	for i := 0; i <= BanknoteChainPLedgerNumber; i++ {
		var ethdb ethdb.Database
		if i == 0 {
			lvldbPath := filepath.Join(cc.StoragePath, "CoreLedgerDB", strconv.FormatUint(cc.NodeID, 10))
			ethdb, _ = rawdb.NewLevelDBDatabase(lvldbPath, cc.CacheSize, 32, "", false)
		} else {
			dbname := fmt.Sprintf("%s%d", "PLedgerStateDB_", i%2)
			lvldbPath := filepath.Join(cc.StoragePath, dbname, strconv.FormatUint(cc.NodeID, 10))
			ethdb, _ = rawdb.NewLevelDBDatabase(lvldbPath, cc.CacheSize, 32, "", false)
		}

		bcs.Ethdbs = append(bcs.Ethdbs, ethdb)
		triedb := trie.NewDatabaseWithConfig(ethdb, &trie.Config{
			Cache:     cc.CacheSize,
			Preimages: true,
		})

		bcs.Triedbs = append(bcs.Triedbs, triedb)
		// check the existence of the trie database
		_, err := trie.New(trie.TrieID(ledgerStateRoots[i]), triedb)
		if err != nil {
			log.Panic(err)
		}
		fmt.Println("The state trie can be built of banknote ledger", i)

	}
	return bcs
}

func NewBanknoteChainStateGenisis(cc *utils.Config, currentBlockNumber uint64) *BanknoteChainState {
	BanknoteChainPLedgerNumber := cc.BanknoteChainPLedgerNumber
	bcs := &BanknoteChainState{
		CurrentBlockNumber: currentBlockNumber,
		PLedgerStateRoots:  nil,
	}

	activeBanknotesPool := NewActiveBanknotePool(cc.BanknoteChainPLedgerNumber)
	bcs.ActiveBanknotesPool = activeBanknotesPool

	PLedgerStateRoots := make([]common.Hash, 0)
	var CLedgerStateRoot common.Hash
	for i := 0; i <= BanknoteChainPLedgerNumber; i++ {
		var ethdb ethdb.Database
		if i == 0 {
			lvldbPath := filepath.Join(cc.StoragePath, "CoreLedgerStateDB", strconv.FormatUint(cc.NodeID, 10))
			ethdb, _ = rawdb.NewLevelDBDatabase(lvldbPath, cc.CacheSize, 32, "", false)
		} else {
			dbname := fmt.Sprintf("%s%d", "PLedgerStateDB_", i)
			lvldbPath := filepath.Join(cc.StoragePath, dbname, strconv.FormatUint(cc.NodeID, 10))
			ethdb, _ = rawdb.NewLevelDBDatabase(lvldbPath, cc.CacheSize, 32, "", false)
		}

		bcs.Ethdbs = append(bcs.Ethdbs, ethdb)
		triedb := trie.NewDatabaseWithConfig(ethdb, &trie.Config{
			Cache:     cc.CacheSize,
			Preimages: true,
		})

		bcs.Triedbs = append(bcs.Triedbs, triedb)

		statusTrie := trie.NewEmpty(triedb)
		if i == 0 {
			CLedgerStateRoot = statusTrie.Hash()
		} else {
			PLedgerStateRoots = append(PLedgerStateRoots, statusTrie.Hash())
		}

		fmt.Println("The state trie can be built of banknote ledger", i)

	}
	bcs.CLedgerStateRoot = CLedgerStateRoot

	bcs.PLedgerStateRoots = PLedgerStateRoots

	bcs.SignSimulator = utils.NewRSASimulator()

	return bcs
}
