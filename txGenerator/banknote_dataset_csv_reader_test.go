package txGenerator

import (
	log "github.com/sirupsen/logrus"
	"testing"
)

func TestCSVBatchReader_ReadBatch(t *testing.T) {
	csvFilePath := "/Volumes/data/banknote-chain-dataset/tx_16_164.csv"

	batchReader, err := NewBanknoteDatasetBatchReader(csvFilePath)
	if err != nil {
		return
	}
	defer batchReader.Close()
	for {
		batchCSV := batchReader.ReadBanknoteDatasetItem()
		if batchCSV == nil {
			log.Info("出现了NIL！")
		}
		//fmt.Println(batchCSV)
	}

}
