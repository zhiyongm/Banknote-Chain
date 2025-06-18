package params

import "math/big"

type ChainConfig struct {
	ChainID           uint64
	NodeID            uint64
	BlockSize         int
	BlockInterval     uint64
	InjectSpeed       uint64
	BlockDBPath       string
	BNDBPath          string
	IsSender          bool
	DBrootPath        string
	BNEnable          bool
	CleanerInterval   uint64
	NormalInterval    uint64
	GeneratorInterval uint64
	VSNum             uint64
	CacheSize         int
}

var (
	Init_Balance, _ = new(big.Int).SetString("100000000000000000000000000000000000000000000", 10)
)
