package utils

import (
	"encoding/csv"
	"io"
	"math"
	"math/big"
	"os"
	"strconv"
)

type CsvReader struct {
	file   *os.File
	reader *csv.Reader
}

func NewCSVReader(filePath string) (*CsvReader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(file)
	return &CsvReader{
		file:   file,
		reader: reader,
	}, nil
}

func (cr *CsvReader) ReadRow() ([]string, error) {
	return cr.reader.Read()
}

func (cr *CsvReader) ReadAll() ([][]string, error) {
	var records [][]string
	for {
		record, err := cr.ReadRow()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func (cr *CsvReader) Reads(num int) ([][]string, error) {
	var records [][]string

	for i := 0; i < num; i++ {
		record, err := cr.reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		records = append(records, record)
	}

	if len(records) == 0 {
		return nil, nil
	}

	return records, nil
}
func ConvertFloatStringToBigInt(s string) *big.Int {

	f, _, err := big.ParseFloat(s, 10, 256, big.ToZero)
	if err != nil {
		return nil
	}

	i := new(big.Int)

	f.Int(i)
	return i
}
func ConvertFloatStringToInt(s string) int {
	f, _ := strconv.ParseFloat(s, 64)
	return int(math.Floor(f))
}
