package network

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/sirupsen/logrus"
)

type P2pNetwork struct {
	Host   *host.Host
	Pubsub *pubsub.PubSub
	Sub    *pubsub.Subscription
	Topic  *pubsub.Topic
}

type discoveryNotifee struct {
	h host.Host
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	logrus.Info("discovered new peer %s\n", pi.ID)
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		return
	}
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", pi.ID, err)
	}
}
func setupDiscovery(h host.Host) error {
	s := mdns.NewMdnsService(h, "CoinpurseNet", &discoveryNotifee{h: h})
	return s.Start()
}

func NewP2PNetwork(isSender bool) *P2pNetwork {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))

	ps, err := pubsub.NewGossipSub(context.Background(), h, pubsub.WithMaxMessageSize(1<<40))

	topic, err := ps.Join("txs")
	var sub *pubsub.Subscription
	if !isSender {
		sub, _ = topic.Subscribe()
	}
	if err != nil {
		panic(err)
	}
	if err := setupDiscovery(h); err != nil {
		panic(err)
	}
	return &P2pNetwork{&h, ps, sub, topic}
}
