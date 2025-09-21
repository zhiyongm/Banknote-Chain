package utils

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
)

// 这个文件是配置文件，包含了配置加载的相关配置和工具函数
type Config struct {
	InitBalance *big.Int
	PublicIP    string
	NodeID      uint64 `json:"node_id"`
	ChainID     uint64 `json:"chain_id"` // 链ID
	//BlockSize     uint64 `json:"block_size"`        // 区块大小
	BlockInterval int `json:"block_interval"` // 区块间隔时间 毫秒
	//InjectSpeed   uint64 `json:"inject_speed"`      // 交易注入速度
	StoragePath  string `json:"storage_root_path"` // 存储路径
	DropDatabase bool   `json:"drop_database"`     // 是否删除数据库
	//InputDatasetPath     string `json:"input_dataset_path"`      // 输入数据集的路径，串行执行
	//InputBanknoteCSVPath string `json:"input_banknote_csv_path"` // 输入钞票数据集的路径,并行执行

	OutputDatasetPath string `json:"output_dataset_path"` // 输出数据集的路径
	CacheSize         int    `json:"cache_size"`          // 缓存大小
	//TxGeneratorMode   string `json:"tx_generator_mode"`   // 交易生成器模式，可能的值有 "random", "csv"
	TxPoolSize int `json:"tx_pool_size"` // 交易池大小

	PeerCount              int    `json:"peer_count"`                // 连接的节点数（除了提议者以外的节点数量）
	ProposerAddress        string `json:"proposer_address"`          // 提议者节点地址
	P2PSenderListenAddress string `json:"p2p_sender_listen_address"` // 提议者节点监听地址
	P2PSenderIDFilePath    string `json:"p2p_sender_id_file_path"`
	P2PTopic               string `json:"p2p_topic"` // P2P主题

	BanknoteChainPLedgerNumber int `json:"banknote_chain_pledger_number"`
	PAccountModifyChanSize     int `json:"p_account_modify_chan_size"`
	WorkNumberPerPLedger       int `json:"work_number_per_pledger"`
	PLedgerInnerChanSize       int `json:"pledger_inner_chan_size"`
	TxCountPerBlock            int `json:"tx_count_per_block"` // 这里的交易是指数据集中的USDT交易，不是模拟出的钞票交易，实际上钞票交易会更多

	ExpMode  string `json:"exp_mode"`  // 实验模式，可能的值有 "origin", "banknote","scalability"
	NodeMode string `json:"node_mode"` // 节点模式，可能的值有 "leader", "peer"

	USDTDatasetCSVPath     string `json:"usdt_dataset_path"`
	BanknoteDatasetCSVPath string `json:"banknote_dataset_path"`
	DebugMode              bool   `json:"debug_mode"`
	ExpBlockNumber         int    `json:"exp_block_number"` // 实验的区块数量
	CoreLimit              int    `json:"core_limit"`       // 允许使用的CPU核心数
}

func NewConfigFromJson(filePath string) *Config {

	if filePath == "" {
		filePath = "config_leader.json"
	}
	Init_Balance, _ := new(big.Int).SetString("100000000000000000000000000000000000000000000", 10)

	config := &Config{
		InitBalance: Init_Balance,
	}
	file, err := os.ReadFile(filePath)
	if err != nil {
		// 如果文件读取失败，打印错误并退出
		panic(fmt.Sprintf("无法读取配置文件 config_leader.json: %v", err))
	}

	// 2. 解析 JSON 数据
	// json.Unmarshal 将 JSON 数据（[]byte类型）解析到指定的结构体指针中
	err = json.Unmarshal(file, &config)
	if err != nil {
		// 如果 JSON 解析失败，打印错误并退出
		panic(fmt.Sprintf("解析 JSON 配置失败: %v", err))
	}

	return config
}
