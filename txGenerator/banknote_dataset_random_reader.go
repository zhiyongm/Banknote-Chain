package txGenerator

import (
	"blockcooker/banknoteChain"
	"blockcooker/utils"
	"math/big"
	"math/rand"
	"time"
)

type BanknoteRandomReader struct {
	randomBanknotes []*banknoteChain.Banknote
}

func (t *BanknoteRandomReader) GetTx() *banknoteChain.BanknoteTx {
	// 从randomBanknotes中随机选择1个banknote作为输入
	randomIndex := rand.Intn(len(t.randomBanknotes))

	bn := t.randomBanknotes[randomIndex]

	// 生成一个新的随机owner地址
	newOwner := utils.RandomAddress()

	return &banknoteChain.BanknoteTx{
		TxHash:         utils.RandomHash(),
		To:             newOwner,
		BanknoteTxType: banknoteChain.BT,
		Amount:         bn.Value,
		BanknoteHash:   bn.BanknoteHash,
	}
}

func NewTxGeneratorFromRandom(blockSize int, config *utils.Config) *BanknoteRandomReader {
	banknoteRandomReader := &BanknoteRandomReader{}
	rand.Seed(time.Now().UnixNano())

	// 生成随机banknotes,数量和blockSize相同
	for i := 0; i < blockSize; i++ {
		bn := banknoteChain.NewBanknote(
			utils.RandomHash(),
			big.NewInt(0),
			utils.RandomAddress(),
			config)
		banknoteRandomReader.randomBanknotes = append(banknoteRandomReader.randomBanknotes, bn)
		//fmt.Println(bn.BanknoteHash.String(), banknoteChain.GetLedgerHashByID(config.BanknoteChainPLedgerNumber, bn.BanknoteHash))
	}

	return banknoteRandomReader
}
