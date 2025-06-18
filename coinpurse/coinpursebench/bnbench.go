package main

import (
	"bufio"
	"coinpurse/chain"
	"coinpurse/core"
	"coinpurse/params"
	"coinpurse/utils"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/cespare/xxhash/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"

	"math/big"
	"math/rand"
	"net/http"

	log "github.com/sirupsen/logrus"

	"os"
	"path"
	"strconv"
	"time"
)

func getCSVReader(csvPath *string) *csv.Reader {
	if csvPath == nil {
		return nil
	}
	fileBlock, _ := os.Open(*csvPath)

	reader := bufio.NewReaderSize(fileBlock, 1024*1024*1024)
	csvReader := csv.NewReader(reader)

	csvReader.Read()

	return csvReader
}

type TxReader struct {
	csvReader       *csv.Reader
	lastBlockNumber int
	batchData       []*core.Transaction
	nextFirstTx     *core.Transaction
	BatchSize       int
	cc              *params.ChainConfig
}

type BNTxReader struct {
	file        *os.File
	reader      *csv.Reader
	current     []*core.BNTxRecord
	batches     []core.BNTxGroup
	batchSize   int
	isEOF       bool
	ShardNumber int
}

func (r *BNTxReader) getNextBNTxsNEW() ([]core.BNTxGroup, error) {

	for len(r.batches) < r.batchSize && !r.isEOF {
		if err := r.readChunks(); err != nil {
			if errors.Is(err, io.EOF) {
				r.isEOF = true

				if len(r.current) > 0 {
					r.batches = append(r.batches, r.current)
					r.current = nil
				}
				break
			}
			return nil, err
		}
	}

	if len(r.batches) == 0 {
		return nil, io.EOF
	}

	returnSize := min(r.batchSize, len(r.batches))
	result := r.batches[:returnSize]
	r.batches = r.batches[returnSize:]

	return result, nil
}

func (r *BNTxReader) readChunks() error {
	for {
		row, err := r.reader.Read()
		if errors.Is(err, io.EOF) {
			return io.EOF
		}
		if err != nil {
			return err
		}

		if len(row) > 0 && row[0] == "~" {

			if len(r.current) > 0 {
				r.batches = append(r.batches, r.current)
				r.current = nil
			}
			return nil
		}

		bnTxRec := &core.BNTxRecord{
			Sender:       common.HexToAddress(row[0]),
			Recipient:    common.HexToAddress(row[1]),
			Amount:       utils.ConvertFloatStringToBigInt(row[2]),
			BanknoteHash: row[4],
			BnValue:      utils.ConvertFloatStringToBigInt(row[5]),

			ShardId:     int(xxhash.Sum64([]byte(row[4])) % uint64(r.ShardNumber)),
			BnNew1Hash:  row[9],
			BnNew1Value: utils.ConvertFloatStringToBigInt(row[10]),
			BnNew2Hash:  row[11],
			BnNew2Value: utils.ConvertFloatStringToBigInt(row[12]),
		}

		bn_type_pure := utils.ConvertFloatStringToInt(row[3])
		if bn_type_pure == 0 {
			bnTxRec.BN_tx_type = core.NormalTx
		}
		if bn_type_pure == 1 {
			bnTxRec.BN_tx_type = core.NewBN
		}
		if bn_type_pure == 2 {
			bnTxRec.BN_tx_type = core.BNTransfer
		}
		if bn_type_pure == 3 {
			bnTxRec.BN_tx_type = core.BNSplit
		}
		if bn_type_pure == 4 {
			bnTxRec.BN_tx_type = core.BNRecycle
		}

		r.current = append(r.current, bnTxRec)

	}
}

func NewBNTxReader(filename string, batchSize int, shardNumber int) (*BNTxReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(file)
	return &BNTxReader{
		file:        file,
		reader:      reader,
		batchSize:   batchSize,
		ShardNumber: shardNumber,
	}, nil
}

func (tr *TxReader) getNextNormalTxs() []*core.Transaction {

	var count = 0
	for {
		record, err := tr.csvReader.Read()
		count++

		if err != nil {

			if err.Error() == "EOF" {
				if len(tr.batchData) > 0 {
					dd := tr.batchData
					tr.batchData = nil
					return dd
				}
				return nil
			}
			return nil
		}

		BlockNumber := 1

		BNValue, _ := strconv.ParseInt(record[3], 10, 64)

		tx := &core.Transaction{
			TxHash:    utils.RandomHash(),
			Sender:    common.HexToAddress(record[1]),
			Recipient: common.HexToAddress(record[2]),
			Value:     big.NewInt(BNValue),
			Nonce:     rand.Uint64(),
			BNID:      "",
			VSNum:     0,
			TxType:    core.NormalTx,
		}

		tr.batchData = append(tr.batchData, tx)

		if count >= tr.BatchSize {

			bd := tr.batchData
			tr.batchData = make([]*core.Transaction, 0)
			tr.lastBlockNumber = BlockNumber

			return bd
		}

	}

}

func NewBNBlockTxReader(csvReader *csv.Reader, cc *params.ChainConfig) *TxReader {
	return &TxReader{
		csvReader:   csvReader,
		batchData:   make([]*core.Transaction, 0),
		BatchSize:   cc.BlockSize,
		cc:          cc,
		nextFirstTx: nil,
	}
}

func main() {

	var bnBlockPATH = flag.String("bnTxPath", "", "bnTxPath")

	var normalTxPath = flag.String("normalTxPath", "", "normalTxPath")

	var SimuFlag = flag.Bool("SimuFlag", false, "Need inputcsv or NO Need")

	var pprofFlag = flag.Bool("PPofFlag", false, "PProf Flag")

	var SimuBNTXNumberPerblock = flag.Int("SimuBNTXNumberPerblock", 100, "SimuAccountTXNumber")

	var SimuNTNumberPerBlock = flag.Int("SimuNTNumberPerBlock", 0, "SimuAccountTXNumber")

	var SimuBlockNumber = flag.Int("SimuBlockNumber", 100, "SimuBlockNumber")

	var SimuAccountTXNumber = flag.Int("SimuAccountTXNumber", 110, "SimuAccountTXNumber")

	var dbPath = flag.String("dbPath", "", "dbPath")

	var VSNUM = flag.Int("shardNumber", 16, "VSNUM")

	var BufferNUM = flag.Int("preBufferBlockCount", 1000, "VSNUM")

	var BufferMaxSize = flag.Int("bufferMaxSize", 100, "VSNUM")

	var blockSize = flag.Int("blockSize", 100, "batchSize")

	var cacheSize = flag.Int("cacheSize", 256, "cacheSize")
	var OutputCSV = flag.String("outputcsv", "bench.csv", "Output CSV file path")
	var CPUnum = flag.Int("CPUNum", 0, "CPUNum")

	flag.Parse()

	if *pprofFlag {

		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	outFile1, err := os.Create(*OutputCSV)
	csvWriter1 := csv.NewWriter(outFile1)
	defer csvWriter1.Flush()
	defer outFile1.Close()

	if err != nil {
		log.Fatalf("无法创建输出CSV文件1: %v", err)
	}

	fmt.Println("preBufferBlockCount", *BufferNUM)
	if *dbPath != "/" && *dbPath != "/*" {
		os.RemoveAll(*dbPath)
	}

	var nodeid = utils.RandomString(5)
	var para = &params.ChainConfig{ChainID: 1, NodeID: 1,
		BlockInterval: 200, DBrootPath: *dbPath,
		InjectSpeed: 100, BlockDBPath: path.Join(*dbPath, "/blockdb_"+nodeid), IsSender: true, VSNum: uint64(*VSNUM),

		BlockSize: *blockSize, CacheSize: *cacheSize}
	var stateDB, _ = rawdb.NewLevelDBDatabase(path.Join(*dbPath, "/statedb_"+nodeid), 128, 32, "", false)

	var bc, _ = chain.NewBlockChain(para, stateDB)

	bc.SimuFlag = *SimuFlag

	if *CPUnum != 0 {
		bc.GoroutineNum = *CPUnum
		runtime.GOMAXPROCS(*CPUnum)
	}

	var normalTxReader *TxReader
	var bnTxReader *BNTxReader
	if *bnBlockPATH != "" {
		bnTxReader, _ = NewBNTxReader(*bnBlockPATH, para.BlockSize, *VSNUM)
	} else {
		bnBlockPATH = nil
	}

	if *normalTxPath != "" {
		normalTxReader = NewBNBlockTxReader(getCSVReader(normalTxPath), para)
	} else {
		normalTxPath = nil
	}

	txs_chan := make(chan *core.TxsInABlock, *BufferMaxSize)
	NO_MORE_TXS := false

	if !*SimuFlag {
		go func() {
			for {

				var normalTxs []*core.Transaction
				var bnTxs []core.BNTxGroup
				wg := sync.WaitGroup{}
				wg.Add(2)
				go func() {
					if bnBlockPATH != nil {
						get_txs, _ := bnTxReader.getNextBNTxsNEW()
						if get_txs != nil {

							bnTxs = append(bnTxs, get_txs...)
						}
					}
					wg.Done()
				}()

				go func() {
					if normalTxPath != nil {
						get_txs := normalTxReader.getNextNormalTxs()

						if get_txs != nil {

							normalTxs = append(normalTxs, get_txs...)
						}
					}
					wg.Done()
				}()

				wg.Wait()

				if len(normalTxs)+len(bnTxs) == 0 {
					fmt.Println("No more txs!!!!!!!!!!!!!!!!!!!!!")
					NO_MORE_TXS = true
					break
				}

				txs_chan <- &core.TxsInABlock{
					NormalTxs:  normalTxs,
					BNTxGroups: bnTxs,
				}
			}
		}()
	} else {

		go func() {
			blockCount := 0
			address_used_count := 0
			var addr_a common.Address
			var addr_b common.Address
			for {
				if blockCount == *SimuBlockNumber {
					fmt.Println("No more txs!!!!!!!!!!!!!!!!!!!!!")
					NO_MORE_TXS = true
					break
				}
				if address_used_count%*SimuAccountTXNumber == 0 {
					addr_a = utils.GenerateEthereumAddress()
					addr_b = utils.GenerateEthereumAddress()
					address_used_count = 0
				} else {
					address_used_count += 1
				}

				address_used_count += 1
				var normalTxs []*core.Transaction
				var bnTxs []core.BNTxGroup

				for i := 0; i < *SimuBNTXNumberPerblock; i++ {
					var bntxgroup core.BNTxGroup
					var bntx = &core.BNTxRecord{}
					bntx.Sender = addr_a
					bntx.Recipient = addr_b
					bntx.Amount = big.NewInt(rand.Int63())
					bntx.BanknoteHash = utils.RandomString(32)
					bntx.BnValue = big.NewInt(rand.Int63())
					bntx.ShardId = rand.Intn(*VSNUM)
					bntx.BN_tx_type = core.BNTransfer
					bntxgroup = append(bntxgroup, bntx)
					bntxgroup = append(bntxgroup, bntx)
					bnTxs = append(bnTxs, bntxgroup)
				}

				for i := 0; i < *SimuNTNumberPerBlock; i++ {
					var normalTx = &core.Transaction{
						TxHash:    utils.RandomHash(),
						Sender:    addr_a,
						Recipient: addr_b,
						Value:     big.NewInt(rand.Int63()),
						Nonce:     rand.Uint64(),
						TxType:    core.NormalTx,
					}
					normalTxs = append(normalTxs, normalTx)
				}

				txs_chan <- &core.TxsInABlock{
					NormalTxs:  normalTxs,
					BNTxGroups: bnTxs,
				}
				blockCount++
			}

		}()

	}

	var count int

	for {

		time.Sleep(1 * time.Second)
		log.Info("Waiting for txs Buffer to fill up", " ", len(txs_chan), "/ ", *BufferNUM)
		if len(txs_chan) >= *BufferNUM {
			fmt.Println("Start to generate blocks~~~~~~~~~~~~~~~~~~~")
			break
		}

	}

	time.Sleep(1 * time.Second)

	var blockprocessed_time time.Duration
	csvWriter1.Write([]string{"Blocknumber", "Time Stamp", "Block Info", "StateDBSize", "BlockDBSize", "VSStateDBSize", "ActiveBNDBSize", "block_speed", "tx_speed", "ShardNumber",
		"Total_Txs_Count", "TotalTime"})

	showData := func(csvWriter1 *csv.Writer, thisBlockProcessTime *time.Duration) {

		stateDBSize, _ := DirSize(path.Join(*dbPath, "/statedb_"+nodeid))
		blockDBSize, _ := DirSize(path.Join(*dbPath, "/blockdb_"+nodeid))

		var vsStateDBsize int64
		var ActiveBNDBSize int64
		for _, stateDBPath := range bc.BNStateDBPaths {
			size, _ := DirSize(stateDBPath)
			vsStateDBsize += size
		}

		fmt.Println("-----------------------------------------")
		fmt.Println("Time Stamp", time.Now())

		fmt.Printf("StateDBSize: %vB , BlockDBSize: %vB, VSStateDBSize: %vB, ActiveBNDBSize: %vB\n", stateDBSize, blockDBSize, vsStateDBsize, ActiveBNDBSize)

		elapsed := blockprocessed_time
		now_tx_count := bc.ProcessedTxCount

		var block_speed float64
		if thisBlockProcessTime != nil {
			block_speed = (float64(*SimuBNTXNumberPerblock) + float64(*SimuNTNumberPerBlock)) / float64(thisBlockProcessTime.Milliseconds()) * 1000
		} else {
			block_speed = 0
		}

		tx_speed := float64(now_tx_count) / elapsed.Seconds()

		fmt.Printf("Block Speed: %.2f iterations per second\n", block_speed)
		fmt.Printf("Tx Speed: %.2f Txs per second\n", tx_speed)
		fmt.Println("Total Txs processed: ", now_tx_count)
		fmt.Println("Latest Block Info: ", bc.CurrentBlock.PrintBlock())
		fmt.Println("Sender Info: ", len(txs_chan), "/", *BufferMaxSize)
		fmt.Println("Total Time Elapsed: ", elapsed.Milliseconds())

		fmt.Println("Shards Count: ", bc.VSNum)
		fmt.Println("-----------------------------------------")

		record := []string{
			strconv.Itoa(int(bc.CurrentBlock.Header.Number)),
			strconv.Itoa(int(time.Now().Unix())),
			bc.CurrentBlock.PrintBlock(),
			strconv.Itoa(int(stateDBSize)),
			strconv.Itoa(int(blockDBSize)),
			strconv.Itoa(int(vsStateDBsize)),
			strconv.Itoa(int(ActiveBNDBSize)),
			strconv.Itoa(int(block_speed)),
			strconv.FormatFloat(tx_speed, 'f', -1, 64),
			strconv.Itoa(int(bc.VSNum)),
			strconv.Itoa(int(now_tx_count)),
			strconv.Itoa(int(elapsed.Milliseconds())),
		}
		csvWriter1.Write(record)
		csvWriter1.Flush()
	}

	showFlag := false

FORMAIN:
	for {
		select {
		case txs := <-txs_chan:
			{
				showFlag = true
				start := time.Now()
				blk := bc.GenerateBlockwithTxs(txs)
				for _, tx := range blk.Body {

					tx.Sig = make([]byte, 65)
				}
				bc.AddBlock(blk)
				dur := time.Since(start)
				blockprocessed_time += dur
				count++
				showData(csvWriter1, &dur)
			}
		default:
			{
				if showFlag {
					if NO_MORE_TXS {
						fmt.Println("Experiment is over!!!")
						showData(csvWriter1, nil)
						break FORMAIN
					}

					fmt.Println("Waiting for txs generation......")
					time.Sleep(2 * time.Second)
					showFlag = false
				}

			}
		}

	}

}

func DirSize(path string) (int64, error) {

	var size int64
	var wg sync.WaitGroup
	fileChan := make(chan string, 100)

	go func() {
		wg.Add(1)
		defer wg.Done()

		filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				fileChan <- filePath
			}
			return nil
		})

		close(fileChan)
	}()

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range fileChan {
				info, err := os.Stat(filePath)
				if err != nil {

					continue
				}
				atomic.AddInt64(&size, info.Size())
			}
		}()
	}

	wg.Wait()
	return size, nil

}

func ConcurrentDirSize(path string) (int64, error) {
	var size int64
	var wg sync.WaitGroup
	fileChan := make(chan string, 100)

	go func() {
		wg.Add(1)
		defer wg.Done()

		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				fileChan <- filePath
			}
			return nil
		})
		if err != nil {
			fmt.Printf("遍历错误: %v\n", err)
		}
		close(fileChan)
	}()

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range fileChan {
				info, err := os.Stat(filePath)
				if err != nil {
					fmt.Printf("文件错误: %v\n", err)
					continue
				}
				atomic.AddInt64(&size, info.Size())
			}
		}()
	}

	wg.Wait()
	return size, nil
}
