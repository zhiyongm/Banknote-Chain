package message

import (
	"coinpurse/core"
	"time"
)

type BlockInfoMsg struct {
	BlockBodyLength int
	AccountTxs      []*core.Transaction

	Banknotes []*core.Transaction

	ProposeTime   time.Time
	CommitTime    time.Time
	SenderShardID uint64

	Relay1Txs []*core.Transaction

	Broker1Txs []*core.Transaction
}

type RawMessage struct {
	Content []byte
}
