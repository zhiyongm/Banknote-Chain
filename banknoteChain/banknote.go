package banknoteChain

import (
	"blockcooker/utils"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type Banknote struct {
	sync.Mutex
	BanknoteHash  common.Hash
	Value         *big.Int
	Owner         common.Address
	LedgerID      int
	TurnOverCount int
}

func NewBanknote(banknoteHash common.Hash, banknoteValue *big.Int, owner common.Address, cc *utils.Config) *Banknote {
	return &Banknote{
		BanknoteHash:  banknoteHash,
		Value:         banknoteValue,
		LedgerID:      GetLedgerHashByID(cc.BanknoteChainPLedgerNumber, banknoteHash),
		Owner:         owner,
		TurnOverCount: 0,
	}

}

func GetLedgerHashByID(PLedgerNumber int, banknoteHash common.Hash) int {
	ledgerID := banknoteHash.Big().Uint64() % uint64(PLedgerNumber)
	return int(ledgerID)
}
