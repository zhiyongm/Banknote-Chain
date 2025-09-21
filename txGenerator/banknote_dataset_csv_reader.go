package txGenerator

import (
	"blockcooker/banknoteChain"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gocarina/gocsv"
	log "github.com/sirupsen/logrus"
)

// 定义你的CSV数据结构
// 字段名需要与CSV文件中的列名匹配
type BanknoteTxCSVRow struct {
	Sender        string `csv:"sender"`
	Recipient     string `csv:"recipient"`
	BN_tx_type    string `csv:"BN_tx_type"`
	BNHash        string `csv:"banknote_hash"`
	BanknoteValue string `csv:"banknote_value"`
	BNNew1Hash    string `csv:"bn_new1_hash"`
	BNNew1Value   string `csv:"newBN1_value"`
	BNNew2Hash    string `csv:"bn_new2_hash"`
	BNNew2Value   string `csv:"newBN2_value"`
	ShardID       int    `csv:"shard_id"`
	Creator       string `csv:"bn_creator"`
	LineInCSV     uint64
}

// BanknoteDatasetBatchReader 是一个自定义的CSV分批读取器
type BanknoteDatasetBatchReader struct {
	reader    *bufio.Reader
	file      *os.File
	isEOF     bool
	countLine uint64
}

// NewCSVBatchReader 创建并返回一个CSVBatchReader对象
// filePath: 要读取的CSV文件路径
func NewBanknoteDatasetBatchReader(filePath string) (*BanknoteDatasetBatchReader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件: %w", err)
	}

	// 使用1MB的缓冲来读取文件，以提高性能
	const bufferSize = 10 * 1024 * 1024
	reader := bufio.NewReaderSize(file, bufferSize)
	reader.ReadString('\n') // 读取并丢弃第一行（CSV头）
	return &BanknoteDatasetBatchReader{
		reader: reader,
		file:   file,
		isEOF:  false,
	}, nil
}

// ReadBatch 读取下一组数据
// 如果读取成功，返回一组数据和一个nil错误
// 如果到达文件末尾，返回nil和nil错误
// 如果发生其他错误，返回nil和相应的错误
func (r *BanknoteDatasetBatchReader) ReadBatch() ([]*BanknoteTxCSVRow, error) {
	if r.isEOF {
		return nil, nil // 文件已读完，返回nil
	}

	var batchData strings.Builder
	var isFirstLineOfBatch = true

	for {
		line, err := r.reader.ReadString('\n')
		r.countLine++

		if err == io.EOF {
			r.isEOF = true
			if batchData.Len() > 0 {
				// 最后一批数据，没有特殊行分隔
				break
			} else {
				// 文件末尾没有数据了
				return nil, nil
			}
		} else if err != nil {
			return nil, fmt.Errorf("读取文件出错: %w", err)
		}

		// 检查是否为特殊行
		if strings.HasPrefix(line, "~") {
			if batchData.Len() > 0 {
				break // 找到下一组的起始，结束当前批次的读取
			}
			// 如果第一行就是分隔符，则跳过
			continue
		}

		// 如果是第一行，我们需要添加CSV头信息
		// 注意: gocsv 要求第一行是CSV头，如果你的文件没有，你需要自己生成一个
		if isFirstLineOfBatch {
			// 这里假设你的CSV文件头是 "id,name,value"，你需要根据实际情况调整
			header := "sender,recipient,amount,BN_tx_type,banknote_hash,banknote_value,shard_id,bn_turnover_count,bn_creator,bn_new1_hash,newBN1_value,bn_new2_hash,newBN2_value\n"
			batchData.WriteString(header)
			isFirstLineOfBatch = false
		}

		// 将当前行添加到批次数据中
		batchData.WriteString(line)
	}

	// 如果没有数据被读取，说明文件已读完或批次为空
	if batchData.Len() == 0 {
		return nil, nil
	}

	// 使用gocsv反序列化批次数据
	var records []*BanknoteTxCSVRow
	csvBytes := bytes.NewReader([]byte(batchData.String()))
	if err := gocsv.Unmarshal(csvBytes, &records); err != nil {
		return nil, fmt.Errorf("gocsv反序列化出错: %w", err)
	}

	for _, record := range records {
		record.LineInCSV = r.countLine
	}

	return records, nil
}

// 每读取一下这里，就返回一组banknoteTx，是一个原始USDT产生的一系列的banknoteTx
func (r *BanknoteDatasetBatchReader) ReadBanknoteDatasetItem() []*banknoteChain.BanknoteTx {
	BanknoteTxCSVRows, _ := r.ReadBatch()
	var banknoteTxs []*banknoteChain.BanknoteTx
	for i := 1; i < len(BanknoteTxCSVRows); i++ {
		//跳过每组的第一行，第一行是原始的USDT行为，不是banknote
		BanknoteTxCSVRow := BanknoteTxCSVRows[i]

		if BanknoteTxCSVRow.BN_tx_type == "0.0" {
			//跳过BN_tx_type为0的行，这些行不是banknote交易
			continue
		}

		//去掉.0
		BanknoteValue := strings.TrimSuffix(BanknoteTxCSVRow.BanknoteValue, ".0")

		amount, _ := new(big.Int).SetString(BanknoteValue, 10)
		if amount == nil {
			log.Panic("amount is nil", BanknoteTxCSVRow.BanknoteValue, BanknoteTxCSVRow, BanknoteTxCSVRow.LineInCSV)
		}

		var amountNew1 *big.Int
		var amountNew2 *big.Int

		if BanknoteTxCSVRow.BNNew1Value != "" {
			amountNew1, _ = new(big.Int).SetString(BanknoteTxCSVRow.BNNew1Value, 10)
		} else {
			amountNew1 = big.NewInt(0)
		}

		if BanknoteTxCSVRow.BNNew2Value != "" {
			amountNew2, _ = new(big.Int).SetString(BanknoteTxCSVRow.BNNew2Value, 10)
		} else {
			amountNew2 = big.NewInt(0)
		}

		var TxType banknoteChain.BanknoteTxType
		if BanknoteTxCSVRow.BN_tx_type == "1.0" {
			TxType = banknoteChain.BG
			//fmt.Println("BG", BanknoteTxCSVRow)

		} else if BanknoteTxCSVRow.BN_tx_type == "2.0" {
			TxType = banknoteChain.BT
			//fmt.Println("BT", BanknoteTxCSVRow)

		} else if BanknoteTxCSVRow.BN_tx_type == "3.0" {
			TxType = banknoteChain.BS
			//fmt.Println("BS", BanknoteTxCSVRow)

		} else if BanknoteTxCSVRow.BN_tx_type == "4.0" {
			TxType = banknoteChain.BR
			//fmt.Println("BR", BanknoteTxCSVRow)

		}

		banknoteTx := banknoteChain.BanknoteTx{
			From:                          common.HexToAddress(BanknoteTxCSVRow.Sender),
			To:                            common.HexToAddress(BanknoteTxCSVRow.Recipient),
			BanknoteHash:                  crypto.Keccak256Hash([]byte(BanknoteTxCSVRow.BNHash)),
			BanknoteOriginIDinCSV:         BanknoteTxCSVRow.BNHash,
			Amount:                        amount,
			BanknoteTxType:                TxType,
			Creator:                       common.HexToAddress(BanknoteTxCSVRow.Creator),
			BanknoteHash1NewForSplit:      crypto.Keccak256Hash([]byte(BanknoteTxCSVRow.BNNew1Hash)),
			BanknoteHash1NewOriginIDinCSV: BanknoteTxCSVRow.BNNew1Hash,
			Amount1NewForSplit:            amountNew1,
			BanknoteHash2NewForSplit:      crypto.Keccak256Hash([]byte(BanknoteTxCSVRow.BNNew2Hash)),
			BanknoteHash2NewOriginIDinCSV: BanknoteTxCSVRow.BNNew2Hash,
			Amount2NewForSplit:            amountNew2,
		}

		banknoteTx.TxHash = crypto.Keccak256Hash(banknoteTx.Encode())

		banknoteTxs = append(banknoteTxs, &banknoteTx)
	}
	if banknoteTxs == nil {
		return nil
	}
	return banknoteTxs
}

// Close 关闭文件句柄
func (r *BanknoteDatasetBatchReader) Close() error {
	return r.file.Close()
}
