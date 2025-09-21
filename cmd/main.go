package main

import (
	"blockcooker/chain"
	"blockcooker/core"
	p2p "blockcooker/network"
	"blockcooker/utils"
	"context"
	"flag"
	_ "flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"net/http"
	_ "net/http/pprof" // 导入这个包会自动注册 pprof 路由
)

func initLog() {

	//// 设置日志格式为json格式
	//log.SetFormatter(&log.TextFormatter{})
	//
	//// 设置将日志输出到标准输出（默认的输出为stderr，标准错误）
	//// 日志消息输出可以是任意的io.writer类型
	//log.SetOutput(os.Stdout)

}
func main() {
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	var configPath = flag.String("c", "config_leader.json", "path to config file")
	flag.Parse()

	config := utils.NewConfigFromJson(*configPath)
	if config.CoreLimit != 0 {
		log.Info("CPU 核心数限制为：", config.CoreLimit)
		runtime.GOMAXPROCS(config.CoreLimit)
	}

	if config.DropDatabase {
		if config.StoragePath != "/" {
			os.RemoveAll(config.StoragePath)
		}
	}

	exitChan := make(chan os.Signal)
	signal.Notify(exitChan, os.Interrupt, os.Kill, syscall.SIGTERM)

	if config.NodeMode == "leader" {
		initLog()
		log.Info("启动Leader节点模式 transaction injector mode")
		go startLeader(ctx, config)
	}

	if config.NodeMode == "peer" {
		initLog()
		log.Info("启动peer节点模式 Peer node mode")
		go startPeer(ctx, config)
	}

	// 无法识别的节点模式，提示错误退出
	if config.NodeMode != "leader" && config.NodeMode != "peer" {
		log.Error("无法识别的节点模式，请检查配置文件中的 node_mode 设置。 Unrecognized node mode, please check the node_mode setting in the configuration file.")
		os.Exit(1)
	}

	if config.DebugMode {
		go func() {
			// 启动一个独立的 HTTP 服务来提供 pprof 信息
			// 生产环境中，最好不要在主服务端口暴露这个路由
			http.ListenAndServe("0.0.0.0:6060", nil)
		}()

	}
	for {
		select {
		case sig := <-exitChan:
			fmt.Println("进程停止：", sig)
			cancel()
			time.Sleep(1 * time.Second)
			os.Exit(0)
		}
	}
}

func startLeader(ctx context.Context, cc *utils.Config) {
	bc, _ := chain.NewBlockChain(cc)
	log.Info("区块链初始化完成，开始交易注入")

	log.Info("获取公网IP地址")
	cc.PublicIP = utils.GetPublicIP()
	log.Info("公网IP地址获取成功：", cc.PublicIP)

	time.Sleep(5 * time.Second) // 等待交易池准备好
	log.Info("交易池准备就绪，开始区块生成")

	log.Info("启动P2P发布者")
	p2pPublisher, err := p2p.NewPublisher(ctx, cc.P2PTopic, cc.P2PSenderIDFilePath, cc.P2PSenderListenAddress)
	if err != nil {
		log.Error("P2P发布者初始化失败，1秒后重试：", err)
		time.Sleep(1 * time.Second)
	}

	for {
		if len(p2pPublisher.Topic.ListPeers()) >= cc.PeerCount {
			log.Info("P2P节点数量达到要求，开始区块生成，当前连接数：", len(p2pPublisher.Topic.ListPeers()), "/", cc.PeerCount)
			break
		} else {
			log.Info("等待P2P连接，当前连接数：", len(p2pPublisher.Topic.ListPeers()), "/", cc.PeerCount)
			time.Sleep(1 * time.Second)
		}
	}

	go bc.StartBlockGenerator()              // 启动区块生成器
	go bc.StartBlockAdder(ctx, p2pPublisher) // 启动区块添加器

}

func startPeer(ctx context.Context, cc *utils.Config) {
	bc, _ := chain.NewBlockChain(cc)
	log.Info("区块链初始化完成，开始同步区块")

	log.Info("获取公网IP地址")
	cc.PublicIP = utils.GetPublicIP()
	log.Info("公网IP地址获取成功：", cc.PublicIP)

	go bc.StartBlockAdder(ctx, nil) // 启动区块添加器

	log.Info("启动P2P接收者")

	var p2pSubscriber *p2p.Subscriber
	for {
		var err error
		p2pSubscriber, err = p2p.NewSubscriber(ctx, cc.P2PTopic, cc.P2PSenderIDFilePath, cc.ProposerAddress)
		if err != nil {
			log.Error("P2P订阅者初始化失败，1秒后重试：", err)
			time.Sleep(1 * time.Second)
			continue
		}
		log.Info("P2P订阅者初始化成功，开始接收区块")
		break
	}

	for {
		log.Info("开始接收区块")
		msg, err := p2pSubscriber.Sub.Next(ctx)
		if err != nil {
			log.Error("接收消息失败", err)
		}

		// 避免接收到自己发送的消息。
		if msg.GetFrom() == p2pSubscriber.Host.ID() {
			continue
		}

		fmt.Printf("Subscriber %s received message from %s Size: %.2f MB \n", p2pSubscriber.Host.ID(), msg.GetFrom(), float64(len(msg.GetData()))/1024/1024)
		msg_arrive_time := time.Now()
		blk := core.DecodeB(msg.GetData())
		blk.Header.MessageDelayTime = msg_arrive_time.Sub(blk.Header.ProposedTime)

		log.Info("Recieved a block No", blk.Header.Number, "ON", time.Now())
		bc.ToBeAddedBlocks <- blk
	}
}
