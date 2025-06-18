package core

import (
	"bytes"
	"encoding/gob"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type CoreModify struct {
	Address common.Address
	Balance *big.Int
	IsAdd   bool
}

func (cm *CoreModify) Encode() []byte {
	var buff bytes.Buffer
	encoder := gob.NewEncoder(&buff)
	encoder.Encode(cm)
	return buff.Bytes()
}
