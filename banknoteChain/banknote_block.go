package banknoteChain

import (
	"bytes"
	"encoding/gob"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
)

type BanknoteBlock struct {
	BlockNumber uint64

	BanknoteBlockHash common.Hash
	TxHash            common.Hash
	CLedgerStateRoot  common.Hash
	PLedgerStateRoots []common.Hash
	C_ledger          []*BanknoteTx
	P_ledgers         [][]*BanknoteTx
	BanknoteTxCount   uint64
}

func (banknoteBlock *BanknoteBlock) Encode() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(banknoteBlock)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

func NewBanknoteBlock(pLedgerNumber int, BanknoteTxsBlockPerLedger [][]*BanknoteTx, NormalTxs []*BanknoteTx,
	BlockNumber uint64, AllTxHash *common.Hash,
	CLedgerStateRoot common.Hash, PLedgerStateRoots []common.Hash) *BanknoteBlock {

	banknoteBlock := &BanknoteBlock{}
	banknoteBlock.P_ledgers = make([][]*BanknoteTx, pLedgerNumber)

	var BanknoteBlockHashData []byte
	BanknoteBlockHashData = append(BanknoteBlockHashData, AllTxHash.Bytes()...)
	BanknoteBlockHashData = append(BanknoteBlockHashData, CLedgerStateRoot.Bytes()...)

	banknoteBlock.BanknoteTxCount += uint64(len(NormalTxs))
	for i := 0; i < pLedgerNumber; i++ {
		BanknoteBlockHashData = append(BanknoteBlockHashData, PLedgerStateRoots[i].Bytes()...)
		banknoteBlock.P_ledgers[i] = BanknoteTxsBlockPerLedger[i]
		banknoteBlock.BanknoteTxCount += uint64(len(BanknoteTxsBlockPerLedger[i]))
	}

	banknoteBlock.BlockNumber = BlockNumber
	banknoteBlock.CLedgerStateRoot = CLedgerStateRoot
	banknoteBlock.PLedgerStateRoots = PLedgerStateRoots
	banknoteBlock.C_ledger = NormalTxs
	banknoteBlock.P_ledgers = BanknoteTxsBlockPerLedger

	banknoteBlock.BanknoteBlockHash = crypto.Keccak256Hash(BanknoteBlockHashData)

	return banknoteBlock
}

func NewGenisisBanknoteBlock(pLedgerNumber int, LedgerStateRoots []common.Hash) *BanknoteBlock {

	bb := &BanknoteBlock{
		BlockNumber:       0,
		PLedgerStateRoots: LedgerStateRoots,
		P_ledgers:         make([][]*BanknoteTx, pLedgerNumber),
	}

	return bb
}
