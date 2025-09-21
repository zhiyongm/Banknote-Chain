package resultWriter

type ResultEntity struct {
	PublicIP    string `csv:"public_ip"`
	Mode        string `csv:"mode"` // "leader" or "peer"
	BlockNumber uint64 `csv:"block_number"`
	BlockHash   string `csv:"block_hash"`
	TxNumber    uint64 `csv:"tx_number"`
	ActionType  string `csv:"action_type"`  // "BlkGenerator" or "BlkAdder"
	ProcessTime int64  `csv:"process_time"` // 处理时间，单位为毫秒
	//ModifyAccountCount int    `csv:"modify_account_count"`
	DelayTime int64   `csv:"delay_time"` // 区块落盘的时间戳，为UnixMilli时间戳
	TPS       float64 `csv:"tps"`
}
