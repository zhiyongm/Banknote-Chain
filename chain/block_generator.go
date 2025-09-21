package chain

import (
	"blockcooker/banknoteChain"
	"blockcooker/core"
	p2p "blockcooker/network"
	"blockcooker/txGenerator"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

func (bc *BlockChain) StartBlockGenerator() {

	block_interval := bc.ChainConfig.BlockInterval

	nowBlkNumber := bc.CurrentBlock.Header.Number

	if bc.ChainConfig.ExpMode == "origin" {
		bc.USDTDatasetReader = txGenerator.NewUSDTDatasetTxGeneratorFromCSV(bc.ChainConfig.USDTDatasetCSVPath)
	}
	if bc.ChainConfig.ExpMode == "banknote" {
		bc.BanknoteDatasetBatchReader, _ = txGenerator.NewBanknoteDatasetBatchReader(bc.ChainConfig.BanknoteDatasetCSVPath)
	}

	if bc.ChainConfig.ExpMode == "scalability" {
		bc.BanknoteRandomReader = txGenerator.NewTxGeneratorFromRandom(bc.ChainConfig.TxCountPerBlock, bc.ChainConfig)
	}

	var TobeProcessedTxsChan = make(chan []*banknoteChain.BanknoteTx, 2)

	go func() {
		for {
			var banknote_txs []*banknoteChain.BanknoteTx

			for i := 0; i < bc.ChainConfig.TxCountPerBlock; i++ {

				if bc.ChainConfig.ExpMode == "origin" {
					tx := bc.USDTDatasetReader.GetTx()
					if tx != nil {
						banknote_txs = append(banknote_txs, bc.USDTDatasetReader.GetTx())
					} else {
						log.Info("USDT数据集交易读取完毕")
					}

				}
				if bc.ChainConfig.ExpMode == "banknote" {
					txs := bc.BanknoteDatasetBatchReader.ReadBanknoteDatasetItem()
					if txs != nil {
						banknote_txs = append(banknote_txs, txs...)
					}
				}

				if bc.ChainConfig.ExpMode == "scalability" {
					tx := bc.BanknoteRandomReader.GetTx()
					if tx != nil {
						banknote_txs = append(banknote_txs, tx)
					}
				}

			}
			TobeProcessedTxsChan <- banknote_txs
		}
	}()

	for {

		Origintxs := make([]*core.Transaction, 0)

		nowBlkNumber++

		if bc.ChainConfig.ExpMode == "origin" {

		}

		if bc.ChainConfig.ExpMode == "banknote" {

		}

		banknote_txs := <-TobeProcessedTxsChan

		blk := bc.GenerateBlockForLeader(Origintxs, banknote_txs, nowBlkNumber)

		bc.ToBeAddedBlocks <- blk
		log.Println("[从交易池构建区块]", "区块号", blk.Header.Number, "区块哈希", hex.EncodeToString(blk.Hash),
			"数据集交易数量", bc.ChainConfig.TxCountPerBlock, "Banknote交易数量", len(banknote_txs))

		if nowBlkNumber >= uint64(bc.ChainConfig.ExpBlockNumber) {
			log.Info("达到预设的区块数量，区块生成器停止工作")
			time.Sleep(10 * time.Second)
			if bc.ChainConfig.CoreLimit != 0 {
				os.Exit(0)
			}

			break
		}
		time.Sleep(time.Duration(block_interval) * time.Millisecond)
	}

}

func (bc *BlockChain) StartBlockAdder(ctx context.Context, p2pPublisher *p2p.Publisher) {

	log.Println("启动区块添加器")
	go func() {
		for {
			blk := <-bc.ToBeAddedBlocks

			bc.AddBlock(blk)
			log.Println("[添加区块到主链]", "区块号", blk.Header.Number, "区块哈希", hex.EncodeToString(blk.Hash), "交易数量", len(blk.Body))
			log.Info("当前ToBeAddedBlocks通道长度", len(bc.ToBeAddedBlocks))
			if p2pPublisher != nil {
				blk.Header.ProposedTime = time.Now()
				err := p2pPublisher.PublishMessage(ctx, string(blk.Encode()))
				if err != nil {
					log.Error("消息广播失败", err)
				} else {
					log.Info("消息已广播")
				}
			} else {
			}
			fmt.Println(bc.PrintBlockChain())
		}
	}()
}
