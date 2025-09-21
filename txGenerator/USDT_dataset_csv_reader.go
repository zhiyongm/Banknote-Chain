package txGenerator

import (
	"blockcooker/banknoteChain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gocarina/gocsv"
	"log"
	"math/big"
	"os"
)

// 对比实验用的，USDT数据集交易生成器，都是NT交易
type USDTDatasetTxGeneratorFromCSV struct {
	csvPath       string
	csvEntityChan chan USDTCSVEntity
	txsChan       chan *banknoteChain.BanknoteTx
}

// GetTxs 从通道中获取指定数量的交易。
// 注意：如果CSV中的交易总数少于 txsCount，此方法会提前返回所有可用的交易。
func (t *USDTDatasetTxGeneratorFromCSV) GetTxs(txsCount int) []*banknoteChain.BanknoteTx {
	var txs []*banknoteChain.BanknoteTx
	for i := 0; i < txsCount; i++ {
		// 使用 "comma, ok" idiom 来安全地从通道接收数据
		// 当 txsChan 被关闭且已空时, ok 会是 false
		tx, ok := <-t.txsChan
		if !ok {
			break // 通道已关闭，没有更多交易了，退出循环
		}
		txs = append(txs, tx)
	}
	return txs
}

func (t *USDTDatasetTxGeneratorFromCSV) GetTx() *banknoteChain.BanknoteTx {

	tx, ok := <-t.txsChan
	//fmt.Println(tx.Sender, tx.Recipient, tx.Value, tx.TxHash)
	if !ok {
		return nil
	}
	return tx
}

func NewUSDTDatasetTxGeneratorFromCSV(csvPath string) *USDTDatasetTxGeneratorFromCSV {
	csvEntityChan := make(chan USDTCSVEntity, 10000)
	txsChan := make(chan *banknoteChain.BanknoteTx, 10000)

	tg := &USDTDatasetTxGeneratorFromCSV{
		csvPath:       csvPath,
		csvEntityChan: csvEntityChan,
		txsChan:       txsChan,
	}

	// 1. CSV 读取 goroutine (生产者)
	go func() {
		// ***关键修改 1:***
		// 在这个 goroutine 退出前，关闭 csvEntityChan。
		// 这是向下一个 goroutine 发出“我已经发送完所有数据”的信号。
		//defer close(tg.csvEntityChan)

		file, err := os.Open(csvPath)
		if err != nil {
			log.Fatalf("无法打开文件: %v", err)
		}
		defer file.Close()

		// gocsv.UnmarshalToChan 会阻塞直到文件读取完毕或发生错误。
		// 因此，一旦这个函数返回，我们就可以安全地关闭通道。
		if err := gocsv.UnmarshalToChan(file, tg.csvEntityChan); err != nil {
			log.Printf("解析CSV时发生错误: %v", err)
		}
	}()

	// 2. 交易转换 goroutine (消费者 & 生产者)
	go func() {
		// ***关键修改 2:***
		// 当这个 goroutine 完成所有转换任务后，关闭 txsChan。
		// 这将信号传递给最终的消费者 (例如 GetTxs 方法)。
		defer close(tg.txsChan)
		// ***关键修改 3:***
		// 使用 for range 循环来遍历通道。
		// 当 csvEntityChan 被关闭并且所有值都被读取后，这个循环会自动结束。
		// 这比无限循环 `for {}` 更简洁和安全。
		for csvEntity := range csvEntityChan {
			b := new(big.Int)
			//b.SetString(csvEntity.Value, 10)
			b.SetString(csvEntity.Value, 10)
			tx := banknoteChain.BanknoteTx{
				From:           common.HexToAddress(csvEntity.From),
				To:             common.HexToAddress(csvEntity.To),
				Amount:         b,
				BanknoteTxType: banknoteChain.NT,
			}
			txsChan <- &tx
		}
	}()
	return tg
}
