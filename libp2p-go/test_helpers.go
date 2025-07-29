package main

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
)

// WaitForConnection waits for two nodes to be connected using channels
func WaitForConnection(ctx context.Context, node1, node2 host.Host, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Check if already connected
	if isConnected(node1, node2.ID()) && isConnected(node2, node1.ID()) {
		return nil
	}

	// Create a channel to signal connection
	connected := make(chan struct{})

	// Set up notification for node1
	notifiee1 := &connectionNotifiee{
		targetPeer: node2.ID(),
		connected:  connected,
	}
	node1.Network().Notify(notifiee1)
	defer node1.Network().StopNotify(notifiee1)

	// Wait for connection or timeout
	select {
	case <-connected:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for connection")
	}
}

// WaitForPeerCount waits until a node has the expected number of peers
func WaitForPeerCount(ctx context.Context, node host.Host, expectedCount int, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			currentCount := len(node.Network().Peers())
			return fmt.Errorf("timeout: expected %d peers, got %d", expectedCount, currentCount)
		case <-ticker.C:
			if len(node.Network().Peers()) >= expectedCount {
				return nil
			}
		}
	}
}

// WaitForDHTValue waits until a value is retrievable from DHT
func WaitForDHTValue(ctx context.Context, dhtNode *dht.IpfsDHT, key string, expectedValue []byte, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for DHT value")
		case <-ticker.C:
			value, err := dhtNode.GetValue(ctx, key)
			if err == nil && string(value) == string(expectedValue) {
				return nil
			}
		}
	}
}

// WaitForProtocolReady waits for a protocol to be available on a peer
func WaitForProtocolReady(ctx context.Context, node host.Host, peerID peer.ID, protocolID string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for protocol %s", protocolID)
		case <-ticker.C:
			protocols, err := node.Peerstore().GetProtocols(peerID)
			if err == nil {
				for _, proto := range protocols {
					if string(proto) == protocolID {
						return nil
					}
				}
			}
		}
	}
}

// WaitForAllConnections waits for all nodes to be connected to each other
func WaitForAllConnections(ctx context.Context, nodes []host.Host, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	expectedConnections := len(nodes) - 1

	for _, node := range nodes {
		err := WaitForPeerCount(ctx, node, expectedConnections, timeout)
		if err != nil {
			return fmt.Errorf("node %s failed to connect to all peers: %w", node.ID(), err)
		}
	}
	return nil
}

// connectionNotifiee implements network.Notifiee to listen for connection events
type connectionNotifiee struct {
	targetPeer peer.ID
	connected  chan struct{}
	notified   bool
}

func (n *connectionNotifiee) Listen(network.Network, multiaddr.Multiaddr)      {}
func (n *connectionNotifiee) ListenClose(network.Network, multiaddr.Multiaddr) {}
func (n *connectionNotifiee) Disconnected(network.Network, network.Conn) {}

func (n *connectionNotifiee) Connected(net network.Network, conn network.Conn) {
	if !n.notified && conn.RemotePeer() == n.targetPeer {
		n.notified = true
		close(n.connected)
	}
}

// isConnected checks if two nodes are connected
func isConnected(node host.Host, peerID peer.ID) bool {
	peers := node.Network().Peers()
	for _, p := range peers {
		if p == peerID {
			return true
		}
	}
	return false
}

// WaitWithCondition polls a condition function until it returns true or times out
func WaitWithCondition(ctx context.Context, condition func() bool, timeout time.Duration, interval time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if condition() {
		return nil
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for condition")
		case <-ticker.C:
			if condition() {
				return nil
			}
		}
	}
} 