package banknoteChain

import (
	"bytes"
	"encoding/gob"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
)

type BanknoteTxType int

const (
	BG BanknoteTxType = iota
	BT
	BR
	BS
	NT
)

type BanknoteTx struct {
	TxHash                        common.Hash
	From                          common.Address
	To                            common.Address
	BanknoteHash                  common.Hash
	BanknoteOriginIDinCSV         string
	Amount                        *big.Int
	BanknoteTxType                BanknoteTxType
	Creator                       common.Address
	BanknoteHash1NewForSplit      common.Hash
	BanknoteHash1NewOriginIDinCSV string
	Amount1NewForSplit            *big.Int

	BanknoteHash2NewForSplit      common.Hash
	BanknoteHash2NewOriginIDinCSV string

	Amount2NewForSplit *big.Int
}

func (bntx *BanknoteTx) Encode() []byte {
	var buff bytes.Buffer
	encoder := gob.NewEncoder(&buff)
	err := encoder.Encode(bntx)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

type TxList []*BanknoteTx

func (t TxList) Len() int {
	return len(t)
}

func (t TxList) Less(i, j int) bool {

	hashI := t[i].TxHash.Big()
	hashJ := t[j].TxHash.Big()

	return hashI.Cmp(hashJ) > 0
}

func (t TxList) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func SortBanknoteTxs(txs []*BanknoteTx) []*BanknoteTx {

	txList := TxList(txs)
	sort.Sort(txList)
	return txList
}
