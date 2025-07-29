package main

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

type NodeConfig struct {
	Port           int
	EnableRelay    bool
	EnableWS       bool
	MaxConnections int
	LowWater       int
	HighWater      int
}

func createNode(ctx context.Context, port int, enableRelay bool) (host.Host, error) {
	return createNodeWithOptions(ctx, port, enableRelay, true) // Enable WebSocket by default
}

func createNodeWithOptions(ctx context.Context, port int, enableRelay bool, enableWS bool) (host.Host, error) {
	logrus.Info("Creating libp2p node...")

	config := &NodeConfig{
		Port:           port,
		EnableRelay:    enableRelay,
		EnableWS:       enableWS,
		MaxConnections: 1000,
		LowWater:       50,
		HighWater:      200,
	}

	// Build listen addresses
	listenAddrs := buildListenAddresses(config.Port, config.EnableWS)

	// Create libp2p host options
	opts := []libp2p.Option{
		// Listen addresses - TCP, QUIC (UDP), and WebSocket
		libp2p.ListenAddrs(listenAddrs...),
		
		// Enable hole punching
		libp2p.EnableHolePunching(),
		
		// Enable AutoNAT for NAT detection
		libp2p.EnableAutoNATv2(),
		
		// Enable relay client for hole punching
		libp2p.EnableRelayService(),
	}

	// Add relay service if enabled
	if enableRelay {
		opts = append(opts, libp2p.EnableRelay())
	}

	// Create the host
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Set up routing (DHT)
	if err := setupRouting(ctx, h); err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to setup routing: %w", err)
	}

	// Set up protocols
	if err := setupProtocols(ctx, h); err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to setup protocols: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"peer_id":    h.ID(),
		"addrs":      h.Addrs(),
		"relay":      enableRelay,
		"websocket":  enableWS,
	}).Info("Node created successfully")

	return h, nil
}

func buildListenAddresses(port int, enableWS bool) []multiaddr.Multiaddr {
	var addrs []multiaddr.Multiaddr

	portStr := "0"
	if port > 0 {
		portStr = fmt.Sprintf("%d", port)
	}

	// TCP addresses
	tcpAddr4, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", portStr))
	tcpAddr6, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip6/::/tcp/%s", portStr))
	
	// QUIC addresses (UDP-based)
	quicAddr4, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/udp/%s/quic-v1", portStr))
	quicAddr6, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip6/::/udp/%s/quic-v1", portStr))

	addrs = append(addrs, tcpAddr4, tcpAddr6, quicAddr4, quicAddr6)

	// Add WebSocket addresses if enabled
	if enableWS {
		wsAddr4, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%s/ws", portStr))
		wsAddr6, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip6/::/tcp/%s/ws", portStr))
		
		// WebSocket Secure addresses
		wssAddr4, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%s/wss", portStr))
		wssAddr6, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip6/::/tcp/%s/wss", portStr))
		
		addrs = append(addrs, wsAddr4, wsAddr6, wssAddr4, wssAddr6)
		
		logrus.WithField("websocket", true).Info("WebSocket transport enabled")
	}

	return addrs
}

func setupRouting(ctx context.Context, h host.Host) error {
	// Create a DHT for routing
	kademliaDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeAuto))
	if err != nil {
		return fmt.Errorf("failed to create DHT: %w", err)
	}

	// Bootstrap the DHT
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	logrus.Info("DHT routing setup complete")
	return nil
}

func setupProtocols(ctx context.Context, h host.Host) error {
	// The protocols are automatically set up by libp2p options
	// Additional custom protocols can be added here
	
	logrus.Info("Protocols setup complete")
	return nil
}
