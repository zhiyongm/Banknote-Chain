package chain

import (
	"blockcooker/banknoteChain"
	"blockcooker/core"
	"blockcooker/utils"
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
	log "github.com/sirupsen/logrus"
)

func ProcessBanknoteTxs(BanknoteTxs []*banknoteChain.BanknoteTx,
	banknoteChainState *banknoteChain.BanknoteChainState, cc *utils.Config, blockNumber uint64) *banknoteChain.BanknoteBlock {

	time_start := time.Now()

	var NormalTxs []*banknoteChain.BanknoteTx
	var BanknoteTxsBlockPerLedger [][]*banknoteChain.BanknoteTx
	var BanknoteTxsActionsPerLedger [][]*banknoteChain.BanknoteTx
	for i := 0; i < cc.BanknoteChainPLedgerNumber; i++ {
		BanknoteTxsBlockPerLedger = append(BanknoteTxsBlockPerLedger, make([]*banknoteChain.BanknoteTx, 0))
		BanknoteTxsActionsPerLedger = append(BanknoteTxsActionsPerLedger, make([]*banknoteChain.BanknoteTx, 0))
	}

	for _, tx := range BanknoteTxs {
		if tx.BanknoteTxType == banknoteChain.NT {
			NormalTxs = append(NormalTxs, tx)
			continue
		}
		ledgerID := banknoteChain.GetLedgerHashByID(cc.BanknoteChainPLedgerNumber, tx.TxHash)
		BanknoteTxsActionsPerLedger[ledgerID] = append(BanknoteTxsActionsPerLedger[ledgerID], tx)

		if tx.BanknoteTxType != banknoteChain.BR {
			BanknoteTxsBlockPerLedger[ledgerID] = append(BanknoteTxsBlockPerLedger[ledgerID], tx)
		}
	}

	log.Info("交易分类完毕, 用时 ", time.Since(time_start).String())
	for i := 0; i < cc.BanknoteChainPLedgerNumber; i++ {
		log.Info("外围账本 ", i, " 交易数量 ", len(BanknoteTxsBlockPerLedger[i]), " 状态修改交易数量 ", len(BanknoteTxsActionsPerLedger[i]))
	}

	var waitGroupAllLedgers sync.WaitGroup = sync.WaitGroup{}

	waitGroupAllLedgers.Add(cc.BanknoteChainPLedgerNumber + 1 + 1 + 1) //外围账本+核心账本+交易排序+交易hash计算工作

	var BanknoteTxsBlockPerLedgerSorted [][]*banknoteChain.BanknoteTx = make([][]*banknoteChain.BanknoteTx, cc.BanknoteChainPLedgerNumber)
	var AllTxHash common.Hash

	go func() {
		time_now := time.Now()
		var sortWaitGroup sync.WaitGroup = sync.WaitGroup{}
		sortWaitGroup.Add(cc.BanknoteChainPLedgerNumber)
		for i := 0; i < cc.BanknoteChainPLedgerNumber; i++ {
			go func() {
				BanknoteTxsBlockPerLedgerSorted[i] = banknoteChain.SortBanknoteTxs(BanknoteTxsBlockPerLedger[i])
				sortWaitGroup.Done()
			}()
		}
		sortWaitGroup.Wait()

		waitGroupAllLedgers.Done()
		log.Info("ProcessBanknoteTxs 交易排序完毕, 用时 ", time.Since(time_now).String())
	}()

	go func() {
		time_now := time.Now()
		var txHashsSUM []*big.Int = make([]*big.Int, cc.BanknoteChainPLedgerNumber)
		var hashWaitGroup sync.WaitGroup = sync.WaitGroup{}
		hashWaitGroup.Add(cc.BanknoteChainPLedgerNumber)

		for i := 0; i < cc.BanknoteChainPLedgerNumber; i++ {
			go func() {
				if txHashsSUM[i] == nil {
					txHashsSUM[i] = big.NewInt(0)
				}
				for _, tx := range BanknoteTxsBlockPerLedger[i] {
					txHashsSUM[i] = txHashsSUM[i].Xor(txHashsSUM[i], tx.TxHash.Big())
				}
				hashWaitGroup.Done()

			}()
		}
		hashWaitGroup.Wait()

		var txHashSUM *big.Int = big.NewInt(0)
		for i := 0; i < cc.BanknoteChainPLedgerNumber; i++ {
			txHashSUM = txHashSUM.Xor(txHashSUM, txHashsSUM[i])
		}
		AllTxHash = crypto.Keccak256Hash(txHashSUM.Bytes())
		log.Info("ProcessBanknoteTxs 计算hash完毕, 用时 ", time.Since(time_now).String())
		waitGroupAllLedgers.Done()
	}()

	go ProcessCoreLedgerTx(NormalTxs, banknoteChainState, cc, &waitGroupAllLedgers)

	for i := 0; i < cc.BanknoteChainPLedgerNumber; i++ {
		go ProcessPeripheralLedgerTx(i, BanknoteTxsActionsPerLedger[i], banknoteChainState, cc, &waitGroupAllLedgers)
	}

	waitGroupAllLedgers.Wait()

	newBanknoteBlock := banknoteChain.NewBanknoteBlock(cc.BanknoteChainPLedgerNumber,
		BanknoteTxsBlockPerLedgerSorted, NormalTxs, blockNumber, &AllTxHash,
		banknoteChainState.CLedgerStateRoot, banknoteChainState.PLedgerStateRoots)

	return newBanknoteBlock

}

func ProcessCoreLedgerTx(txs []*banknoteChain.BanknoteTx, banknoteChainState *banknoteChain.BanknoteChainState,
	cc *utils.Config, waitGroupAllLedgers *sync.WaitGroup) (common.Hash, int) {
	time_now := time.Now()

	modified_account_map := make(map[common.Address]bool)
	NowLedgerStateRoot := banknoteChainState.CLedgerStateRoot
	NowLedgerTriedb := banknoteChainState.Triedbs[0]
	st, err := trie.New(trie.TrieID(NowLedgerStateRoot), NowLedgerTriedb)

	if err != nil {
		log.Panic(err)
	}

	cnt := 0
	for _, tx := range txs {
		banknoteChainState.SignSimulator.SimulateVerify()

		s_state_enc, _ := st.Get([]byte(tx.From.Bytes()))
		var s_state *core.AccountState
		if s_state_enc == nil {
			ib := new(big.Int)
			ib.Add(ib, cc.InitBalance)
			s_state = &core.AccountState{
				Nonce:   0,
				Balance: ib,
			}
		} else {
			s_state = core.DecodeAS(s_state_enc)
		}
		modified_account_map[tx.From] = true
		s_state.Deduct(tx.Amount)
		s_state.Nonce++
		st.Update(tx.From.Bytes(), s_state.Encode())
		r_state_enc, _ := st.Get(tx.To.Bytes())
		var r_state *core.AccountState
		if r_state_enc == nil {
			ib := new(big.Int)
			ib.Add(ib, cc.InitBalance)
			r_state = &core.AccountState{
				Nonce:   0,
				Balance: ib,
			}
		} else {
			r_state = core.DecodeAS(r_state_enc)
		}
		modified_account_map[tx.To] = true
		r_state.Deposit(tx.Amount)
		r_state.Nonce++
		st.Update(tx.To.Bytes(), r_state.Encode())
		cnt++
	}
	if cnt == 0 {
		waitGroupAllLedgers.Done()
		log.Info("核心账本 NT 交易处理完毕, 用时 ", time.Since(time_now).String())

		return NowLedgerStateRoot, 0
	}

	rt, ns := st.Commit(false)
	if ns != nil {
		err = NowLedgerTriedb.Update(trie.NewWithNodeSet(ns))
		if err != nil {
			log.Panic()
		}
		err = NowLedgerTriedb.Commit(rt, false)
		if err != nil {
			log.Panic(err)
		}
	}

	banknoteChainState.CLedgerStateRoot = rt

	waitGroupAllLedgers.Done()

	log.Info("核心账本 NT 交易处理完毕, 用时 ", time.Since(time_now).String())
	return rt, len(modified_account_map)
}

func ProcessPeripheralLedgerTx(ledger_id int, txs []*banknoteChain.BanknoteTx,
	banknoteChainState *banknoteChain.BanknoteChainState, cc *utils.Config,
	waitGroupAllLedgers *sync.WaitGroup) {

	time_start := time.Now()
	time_start_all := time.Now()

	NowLedgerStateRoot := banknoteChainState.PLedgerStateRoots[ledger_id]
	NowLedgerTriedb := banknoteChainState.Triedbs[ledger_id+1]
	st, err := trie.New(trie.TrieID(NowLedgerStateRoot), NowLedgerTriedb)
	if err != nil {
		log.Panic(err)
	}
	log.Info("ProcessPeripheralLedgerTx 外围账本 ", ledger_id, " 获取状态树,耗时", time.Since(time_start).String())

	var accountStateModifyChan = make(chan *banknoteChain.BanknoteTx, cc.PAccountModifyChanSize)

	var okChanPStateModify chan bool = make(chan bool)
	go ProcessPeripheralLedgerTxStateTrie(ledger_id, st, NowLedgerTriedb, accountStateModifyChan, banknoteChainState, cc, okChanPStateModify)

	time_create_chan := time.Now()
	var BGTxsChan = make(chan *banknoteChain.BanknoteTx, cc.PLedgerInnerChanSize)
	var BTtxsChan = make(chan *banknoteChain.BanknoteTx, cc.PLedgerInnerChanSize)
	var BStxsChan = make(chan *banknoteChain.BanknoteTx, cc.PLedgerInnerChanSize)
	var BRtxsChan = make(chan *banknoteChain.BanknoteTx, cc.PLedgerInnerChanSize)
	log.Info("ProcessPeripheralLedgerTx 外围账本 ", ledger_id, " 创建交易分类通道,耗时", time.Since(time_create_chan).String())

	go func() {
		time_start := time.Now()
		for _, tx := range txs {
			if tx.BanknoteTxType == banknoteChain.BG {
				BGTxsChan <- tx
			} else if tx.BanknoteTxType == banknoteChain.BT {
				BTtxsChan <- tx
			} else if tx.BanknoteTxType == banknoteChain.BS {
				BStxsChan <- tx
			} else if tx.BanknoteTxType == banknoteChain.BR {
				BRtxsChan <- tx
			}
		}
		close(BGTxsChan)
		close(BTtxsChan)
		close(BStxsChan)
		close(BRtxsChan)
		log.Info("ProcessPeripheralLedgerTx 外围账本 ", ledger_id, " 交易分类完毕,耗时", time.Since(time_start).String())
	}()

	time_start = time.Now()
	var wg sync.WaitGroup

	if cc.CoreLimit == 0 {
		cc.WorkNumberPerPLedger = runtime.NumCPU() / cc.BanknoteChainPLedgerNumber

	} else {
		cc.WorkNumberPerPLedger = cc.CoreLimit / cc.BanknoteChainPLedgerNumber
	}

	if cc.WorkNumberPerPLedger == 0 {
		cc.WorkNumberPerPLedger = 1
	}
	cc.WorkNumberPerPLedger = cc.WorkNumberPerPLedger * 2

	wg.Add(cc.WorkNumberPerPLedger)
	for i := 0; i < cc.WorkNumberPerPLedger; i++ {
		go func() {
			for tx := range BGTxsChan {
				banknoteChainState.SignSimulator.SimulateVerify()
				accountStateModifyChan <- &banknoteChain.BanknoteTx{
					From:   tx.From,
					Amount: tx.Amount,
				}

				banknote := banknoteChain.NewBanknote(tx.BanknoteHash, tx.Amount, tx.From, cc)

				banknoteChainState.ActiveBanknotesPool.AddBanknote(banknote)

			}
			wg.Done()
		}()
	}
	wg.Wait()
	log.Info("外围账本 ", ledger_id, " BG交易处理完毕，等待其他账本,耗时", time.Since(time_start).String())
	time_start = time.Now()
	log.Info("外围账本 ", ledger_id, " 开始处理BS交易")

	toBeRecycledBSSourceBanknotesChan := make(chan *common.Hash, cc.PLedgerInnerChanSize)

	wg.Add(cc.WorkNumberPerPLedger)
	for i := 0; i < cc.WorkNumberPerPLedger; i++ {
		go func() {
			for tx := range BStxsChan {
				banknoteChainState.SignSimulator.SimulateVerify()

				banknote := banknoteChainState.ActiveBanknotesPool.GetBanknote(tx.BanknoteHash)

				if banknote == nil {
					toBeRecycledBSSourceBanknotesChan <- &tx.BanknoteHash
				} else {
					banknoteChainState.ActiveBanknotesPool.RemoveBanknote(banknote.BanknoteHash)
				}

				newBanknote1 := banknoteChain.NewBanknote(tx.BanknoteHash1NewForSplit, tx.Amount1NewForSplit, tx.From, cc)
				newBanknote2 := banknoteChain.NewBanknote(tx.BanknoteHash2NewForSplit, tx.Amount2NewForSplit, tx.To, cc)

				banknoteChainState.ActiveBanknotesPool.AddBanknote(newBanknote1)
				banknoteChainState.ActiveBanknotesPool.AddBanknote(newBanknote2)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	close(toBeRecycledBSSourceBanknotesChan)
	log.Info("外围账本 ", ledger_id, " BS交易处理完毕，等待其他账本,耗时", time.Since(time_start).String())

	time_start = time.Now()

	wg.Add(cc.WorkNumberPerPLedger)
	for i := 0; i < cc.WorkNumberPerPLedger; i++ {
		go func() {
			for tx := range BTtxsChan {
				banknoteChainState.SignSimulator.SimulateVerify()
				banknote := banknoteChainState.ActiveBanknotesPool.GetBanknote(tx.BanknoteHash)
				if banknote == nil {
					amountToCompare := big.NewInt(100000000)
					isInitBanknote := tx.Amount.Cmp(amountToCompare) == 0
					if isInitBanknote {
						banknote = banknoteChain.NewBanknote(tx.BanknoteHash, tx.Amount, tx.To, cc)
						banknoteChainState.ActiveBanknotesPool.AddBanknote(banknote)
					} else {

						continue
					}
				} else {
				}

				banknote.Owner = tx.To
				banknoteChainState.ActiveBanknotesPool.SetBanknote(banknote)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	log.Info("外围账本 ", ledger_id, " BT交易处理完毕，等待其他账本,耗时", time.Since(time_start).String())
	time_start = time.Now()

	wg.Add(cc.WorkNumberPerPLedger)
	for i := 0; i < cc.WorkNumberPerPLedger; i++ {
		go func() {
			for tx := range BRtxsChan {

				accountStateModifyChan <- &banknoteChain.BanknoteTx{
					To:     tx.To,
					Amount: tx.Amount,
				}
				banknote := banknoteChainState.ActiveBanknotesPool.GetBanknote(tx.BanknoteHash)
				if banknote == nil {
					continue
				}
				banknoteChainState.ActiveBanknotesPool.RemoveBanknote(banknote.BanknoteHash)
			}
			wg.Done()
		}()

		for bnHash := range toBeRecycledBSSourceBanknotesChan {
			banknoteChainState.ActiveBanknotesPool.RemoveBanknote(*bnHash)
		}
	}
	wg.Wait()

	log.Info("外围账本", ledger_id, "BR交易处理完毕，等待其他账本,耗时", time.Since(time_start).String())

	time_start = time.Now()

	close(accountStateModifyChan)
	<-okChanPStateModify
	waitGroupAllLedgers.Done()

	log.Info("ProcessBanknoteTxs 外围账本 ", ledger_id, " 处理完毕", " 用时 ", time.Since(time_start_all).String())

}

func ProcessPeripheralLedgerTxStateTrie(ledger_id int,
	st *trie.Trie, triedb *trie.Database, accountStateModifyChan chan *banknoteChain.BanknoteTx,
	banknoteChainState *banknoteChain.BanknoteChainState,
	cc *utils.Config, okChan chan bool) {

	start_time := time.Now()

	accountModifySet := make(map[common.Address]*big.Int)
	for accountStateModifyRecord := range accountStateModifyChan {
		if accountStateModifyRecord.From != *new(common.Address) {
			if am, exists := accountModifySet[accountStateModifyRecord.From]; !exists {
				if accountStateModifyRecord.Amount == nil {
					accountStateModifyRecord.Amount = big.NewInt(0)
				}
				accountModifySet[accountStateModifyRecord.From] = new(big.Int).Sub(big.NewInt(0), accountStateModifyRecord.Amount)
			} else {
				accountModifySet[accountStateModifyRecord.From] = new(big.Int).Sub(am, accountStateModifyRecord.Amount)
			}
		}
		if accountStateModifyRecord.To != *new(common.Address) {
			if am, exists := accountModifySet[accountStateModifyRecord.To]; !exists {
				accountModifySet[accountStateModifyRecord.To] = new(big.Int).Add(big.NewInt(0), accountStateModifyRecord.Amount)
			} else {
				accountModifySet[accountStateModifyRecord.To] = new(big.Int).Add(am, accountStateModifyRecord.Amount)
			}
		}
	}

	cnt := 0
	for addr, amount := range accountModifySet {
		s_state_enc, _ := st.Get([]byte(addr.Bytes()))
		var s_state *core.AccountState
		if s_state_enc == nil {
			ib := new(big.Int)
			ib.Add(ib, cc.InitBalance)
			s_state = &core.AccountState{
				Nonce:   0,
				Balance: ib,
			}
		} else {
			s_state = core.DecodeAS(s_state_enc)
		}

		if amount.Cmp(big.NewInt(0)) < 0 {
			s_state.Deduct(new(big.Int).Abs(amount))
		} else {
			s_state.Deposit(amount)
		}
		s_state.Nonce++
		st.Update(addr.Bytes(), s_state.Encode())
		cnt++
	}

	if cnt == 0 {
		log.Info("ProcessPeripheralLedgerTxStateTrie 外围账本 ", ledger_id, " 状态树更新完毕, 用时 ", time.Since(start_time).String())
		okChan <- true
		return
	}

	rt, ns := st.Commit(false)
	if ns != nil {
		err := triedb.Update(trie.NewWithNodeSet(ns))
		if err != nil {
			log.Panic()
		}
		err = triedb.Commit(rt, false)
		if err != nil {
			log.Panic(err)
		}
	}

	banknoteChainState.PLedgerStateRoots[ledger_id] = rt
	log.Info("ProcessPeripheralLedgerTxStateTrie 外围账本 ", ledger_id, " 状态树更新完毕, 用时 ", time.Since(start_time).String())
	okChan <- true
}
