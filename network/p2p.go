package p2p

import (
	"context"
	"fmt"
	"github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
	"os"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Publisher 结构体封装了用于发布消息的 libp2p 实例和 GossipSub 主题。
type Publisher struct {
	host  host.Host
	ps    *pubsub.PubSub
	Topic *pubsub.Topic
}

// Subscriber 结构体封装了用于订阅和接收消息的 libp2p 实例和 GossipSub 主题订阅。
type Subscriber struct {
	Host host.Host
	ps   *pubsub.PubSub
	Sub  *pubsub.Subscription
}

// loadOrCreatePrivateKey 从指定文件加载私钥，如果文件不存在则创建一个新私钥并保存。
func loadOrCreatePrivateKey(keyFilePath string) (crypto.PrivKey, error) {
	if _, err := os.Stat(keyFilePath); os.IsNotExist(err) {
		// 文件不存在，生成新私钥并保存
		privKey, _, err := crypto.GenerateEd25519Key(nil)
		log.Info("Private key file not found, generating a new one.")
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %w", err)
		}

		keyBytes, err := crypto.MarshalPrivateKey(privKey)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal private key: %w", err)
		}

		if err := os.WriteFile(keyFilePath, keyBytes, 0600); err != nil {
			return nil, fmt.Errorf("failed to write private key to file: %w", err)
		}
		log.Printf("Generated and saved new private key to: %s", keyFilePath)
		return privKey, nil
	}

	// 文件已存在，加载私钥
	keyBytes, err := os.ReadFile(keyFilePath)
	log.Info("Private key file found.")
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	privKey, err := crypto.UnmarshalPrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal private key: %w", err)
	}

	log.Printf("Loaded existing private key from: %s", keyFilePath)
	return privKey, nil
}

// NewPublisher 创建并初始化一个 Publisher 实例。
// 它会启动一个 libp2p 节点并加入指定的话题。
func NewPublisher(ctx context.Context, topicName, keyFilePath string, P2PSenderListenAddress string) (*Publisher, error) {
	privKey, err := loadOrCreatePrivateKey(keyFilePath)
	if err != nil {
		return nil, err
	}

	maxMessageSize := 100 * 1024 * 1024 // 100 MB in bytes

	// 使用固定的私钥创建 libp2p 节点，以确保固定的 Peer ID。
	h, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(P2PSenderListenAddress),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher host: %w", err)
	}

	// 使用 GossipSub 协议初始化 PubSub 服务。
	ps, err := pubsub.NewGossipSub(ctx, h, pubsub.WithMaxMessageSize(maxMessageSize))
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub service: %w", err)
	}

	// 加入指定的话题。
	topic, err := ps.Join(topicName)
	if err != nil {
		return nil, fmt.Errorf("failed to join Topic: %w", err)
	}

	return &Publisher{
		host:  h,
		ps:    ps,
		Topic: topic,
	}, nil
}

// PublishMessage 发送消息到话题。
func (p *Publisher) PublishMessage(ctx context.Context, message string) error {
	//log.Printf("Publisher %s is publishing message: %s", p.host.ID(), message)
	return p.Topic.Publish(ctx, []byte(message))
}

// NewSubscriber 创建并初始化一个 Subscriber 实例。
// 它会启动一个 libp2p 节点，加入指定的话题并开始订阅。
func NewSubscriber(ctx context.Context, topicName string, keyFilePath string, publisher_IP string) (*Subscriber, error) {

	PublisherPrivKey, err := loadOrCreatePrivateKey(keyFilePath)
	publisher_h, err := libp2p.New(
		libp2p.Identity(PublisherPrivKey),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
	)

	Addrs, _ := multiaddr.NewMultiaddr(publisher_IP)

	publisher_Info := peer.AddrInfo{
		ID:    publisher_h.ID(),
		Addrs: []multiaddr.Multiaddr{Addrs},
	}

	if err != nil {
		return nil, err
	}

	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber host: %w", err)
	}
	maxMessageSize := 10000 * 1024 * 1024 // 10000 MB in bytes
	ps, err := pubsub.NewGossipSub(ctx, h, pubsub.WithMaxMessageSize(maxMessageSize))
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub service: %w", err)
	}

	// 加入指定的话题。
	topic, err := ps.Join(topicName)
	if err != nil {
		return nil, fmt.Errorf("failed to join Topic: %w", err)
	}

	// 创建一个订阅，准备接收消息。
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to Topic: %w", err)
	}

	err = ConnectToPeer(ctx, h, publisher_Info)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to publisher: %w", err)
	}
	return &Subscriber{
		Host: h,
		ps:   ps,
		Sub:  sub,
	}, nil
}

// getPublisherAddrInfo 返回 Publisher 的地址信息（包含 IP 和 PeerID），供 Subscriber 连接。
func getPublisherAddrInfo(h host.Host) peer.AddrInfo {
	return peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}
}

// connectToPeer 尝试连接到指定的 peer。
func ConnectToPeer(ctx context.Context, h host.Host, pAddr peer.AddrInfo) error {
	log.Printf("Node %s connecting to peer %s...", h.ID(), pAddr.ID)
	err := h.Connect(ctx, pAddr)
	if err != nil {
		log.Printf("Connection to peer failed: %v", err)
		return err
	} else {
		log.Printf("Connection to peer successful.")
		return nil
	}
}
