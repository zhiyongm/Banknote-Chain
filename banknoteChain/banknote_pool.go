package banknoteChain

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type ActiveBanknotePool struct {
	PoolSize                   int64
	banknotesMaps              []*sync.Map
	BanknoteChainPLedgerNumber int
}

func NewActiveBanknotePool(BanknoteChainPLedgerNumber int) *ActiveBanknotePool {
	abp := &ActiveBanknotePool{
		PoolSize:                   0,
		banknotesMaps:              make([]*sync.Map, BanknoteChainPLedgerNumber),
		BanknoteChainPLedgerNumber: BanknoteChainPLedgerNumber,
	}
	for i := 0; i < BanknoteChainPLedgerNumber; i++ {
		abp.banknotesMaps[i] = &sync.Map{}
	}
	return abp
}

func (bp *ActiveBanknotePool) AddBanknote(banknote *Banknote) {
	banknoteLedgerID := GetLedgerHashByID(bp.BanknoteChainPLedgerNumber, banknote.BanknoteHash)
	banknotesPool := bp.banknotesMaps[banknoteLedgerID]
	banknotesPool.Store(banknote.BanknoteHash, banknote)
}

func (bp *ActiveBanknotePool) SetBanknote(banknote *Banknote) {
	banknoteLedgerID := GetLedgerHashByID(bp.BanknoteChainPLedgerNumber, banknote.BanknoteHash)
	banknotesPool := bp.banknotesMaps[banknoteLedgerID]
	banknotesPool.Store(banknote.BanknoteHash, banknote)
}

func (bp *ActiveBanknotePool) RemoveBanknote(banknoteHash common.Hash) {
	banknoteLedgerID := GetLedgerHashByID(bp.BanknoteChainPLedgerNumber, banknoteHash)
	banknotesPool := bp.banknotesMaps[banknoteLedgerID]

	banknotesPool.Delete(banknoteHash)
}

func (bp *ActiveBanknotePool) GetBanknote(banknoteHash common.Hash) *Banknote {
	banknoteLedgerID := GetLedgerHashByID(bp.BanknoteChainPLedgerNumber, banknoteHash)
	banknotesPool := bp.banknotesMaps[banknoteLedgerID]
	banknote, exist := banknotesPool.Load(banknoteHash)
	if !exist {
		return nil
	} else {
		return banknote.(*Banknote)
	}

}
