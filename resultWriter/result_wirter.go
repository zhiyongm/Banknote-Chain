package resultWriter

import (
	"encoding/csv"
	"fmt"
	"github.com/gocarina/gocsv"
	"os"
	"time"
)

type ResultWriter struct {
	csvPath       string
	CsvEntityChan chan interface{}
	isComplete    chan bool
}

func NewResultWriter(csvPath string) *ResultWriter {
	csvEntityChan := make(chan interface{}, 100000)
	isComplete := make(chan bool, 1)
	file, err := os.Create(csvPath)
	if err != nil {
		fmt.Errorf("无法创建文件 %s: %w", csvPath, err)
	}

	csvWriter := csv.NewWriter(file)
	safeWriter := gocsv.NewSafeCSVWriter(csvWriter)

	rw := &ResultWriter{
		csvPath:       csvPath,
		CsvEntityChan: csvEntityChan,
		isComplete:    isComplete,
	}

	// 启动一个 goroutine 来处理结果写入
	go func() {
		err := gocsv.MarshalChan(csvEntityChan, safeWriter)
		if err != nil && err != gocsv.ErrChannelIsClosed {
			fmt.Printf("写入CSV时发生错误: %v\n", err)
		}
		fmt.Println("✅ CSV数据写入完成。")
		isComplete <- true
	}()

	//设置一个定时器，每5s写入磁盘
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				safeWriter.Flush()
				csvWriter.Flush()
			case <-isComplete:
				safeWriter.Flush()
				csvWriter.Flush()
				return
			}
		}
	}()

	return rw
}
