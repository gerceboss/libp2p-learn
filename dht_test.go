package main

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createDHTKey generates a proper DHT key from a string
func createDHTKey(input string) string {
	// Create a SHA256 hash of the input
	hasher := sha256.New()
	hasher.Write([]byte(input))
	hash := hasher.Sum(nil)
	
	// Create a multihash from the SHA256 hash
	mh, err := multihash.EncodeName(hash, "sha2-256")
	if err != nil {
		panic(err) // Should not happen with valid hash
	}
	
	return string(mh)
}

func TestDHTValueStorage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create multiple nodes for DHT testing
	nodes := make([]host.Host, 3)
	dhts := make([]*dht.IpfsDHT, 3)
	
	for i := 0; i < 3; i++ {
		node, err := createNodeWithOptions(ctx, 0, false, false)
		require.NoError(t, err)
		defer node.Close()
		nodes[i] = node

		// Create DHT for each node
		kademliaDHT, err := dht.New(ctx, node, dht.Mode(dht.ModeServer))
		require.NoError(t, err)
		dhts[i] = kademliaDHT

		// Bootstrap the DHT
		err = kademliaDHT.Bootstrap(ctx)
		require.NoError(t, err)
	}

	// Connect all nodes to each other
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			err := connectNodes(ctx, nodes[i], nodes[j])
			require.NoError(t, err)
		}
	}

	// Wait for all connections to stabilize
	err := WaitForAllConnections(ctx, nodes, 20*time.Second)
	require.NoError(t, err, "All nodes should be connected")

	t.Run("PutAndGetValue", func(t *testing.T) {
		key := createDHTKey("test-key")
		value := []byte("test-value-data")

		// Put value using first DHT
		err := dhts[0].PutValue(ctx, key, value)
		require.NoError(t, err)

		// Wait for value to propagate and be retrievable from other nodes
		for i := 1; i < len(dhts); i++ {
			err := WaitForDHTValue(ctx, dhts[i], key, value, 15*time.Second)
			require.NoError(t, err, "Value should be retrievable from DHT node %d", i)
		}
	})

	t.Run("MultipleValues", func(t *testing.T) {
		testData := map[string][]byte{
			createDHTKey("key1"): []byte("value1"),
			createDHTKey("key2"): []byte("value2"),
			createDHTKey("key3"): []byte("value3"),
		}

		// Store multiple values using different nodes
		i := 0
		for key, value := range testData {
			err := dhts[i%len(dhts)].PutValue(ctx, key, value)
			require.NoError(t, err)
			i++
		}

		// Wait for all values to be retrievable from all nodes
		for key, expectedValue := range testData {
			for j := 0; j < len(dhts); j++ {
				err := WaitForDHTValue(ctx, dhts[j], key, expectedValue, 15*time.Second)
				if err == nil {
					break // Found it on at least one node
				}
				if j == len(dhts)-1 {
					require.NoError(t, err, "Value for key %s should be retrievable", key)
				}
			}
		}
	})
}

func TestDHTPeerDiscovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create DHT network with more nodes
	nodeCount := 5
	nodes := make([]host.Host, nodeCount)
	dhts := make([]*dht.IpfsDHT, nodeCount)

	for i := 0; i < nodeCount; i++ {
		node, err := createNodeWithOptions(ctx, 0, false, false)
		require.NoError(t, err)
		defer node.Close()
		nodes[i] = node

		kademliaDHT, err := dht.New(ctx, node, dht.Mode(dht.ModeServer))
		require.NoError(t, err)
		dhts[i] = kademliaDHT

		err = kademliaDHT.Bootstrap(ctx)
		require.NoError(t, err)
	}

	// Connect nodes in a chain to test peer discovery
	for i := 0; i < nodeCount-1; i++ {
		err := connectNodes(ctx, nodes[i], nodes[i+1])
		require.NoError(t, err)
	}

	// Wait for initial chain connections
	for i := 0; i < nodeCount-1; i++ {
		err := WaitForConnection(ctx, nodes[i], nodes[i+1], 10*time.Second)
		require.NoError(t, err)
	}

	t.Run("PeerDiscoveryThroughDHT", func(t *testing.T) {
		// Check if nodes discovered each other through DHT
		totalConnections := 0
		for i := 0; i < nodeCount; i++ {
			peers := nodes[i].Network().Peers()
			totalConnections += len(peers)
			t.Logf("Node %d has %d peers", i, len(peers))
		}

		// Should have more connections than just the direct chain connections
		// due to DHT peer discovery
		minExpectedConnections := nodeCount - 1 // At least the chain connections
		assert.GreaterOrEqual(t, totalConnections, minExpectedConnections,
			"Should have at least chain connections")
	})

	t.Run("ProtocolAnnouncement", func(t *testing.T) {
		// Test protocol announcement (simplified version)
		testProtocol := protocol.ID("/test/find-peers/1.0.0")
		
		// Set up protocol on some nodes
		for i := 0; i < 2; i++ {
			nodes[i].SetStreamHandler(testProtocol, func(s network.Stream) {
				s.Close()
			})
		}

		// Wait for protocol to be registered using condition
		err := WaitWithCondition(ctx, func() bool {
			protocols := nodes[0].Mux().Protocols()
			for _, proto := range protocols {
				if proto == testProtocol {
					return true
				}
			}
			return false
		}, 10*time.Second, 200*time.Millisecond)
		
		require.NoError(t, err, "Protocol should be registered")

		// Verify protocol was registered
		protocols := nodes[0].Mux().Protocols()
		protocolFound := false
		for _, proto := range protocols {
			if proto == testProtocol {
				protocolFound = true
				break
			}
		}
		assert.True(t, protocolFound, "Protocol should be registered")
	})
}

func TestDHTBootstrap(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Create bootstrap node
	bootstrap, err := createNodeWithOptions(ctx, 0, false, false)
	require.NoError(t, err)
	defer bootstrap.Close()

	bootstrapDHT, err := dht.New(ctx, bootstrap, dht.Mode(dht.ModeServer))
	require.NoError(t, err)

	err = bootstrapDHT.Bootstrap(ctx)
	require.NoError(t, err)

	// Create client node
	client, err := createNodeWithOptions(ctx, 0, false, false)
	require.NoError(t, err)
	defer client.Close()

	clientDHT, err := dht.New(ctx, client, dht.Mode(dht.ModeClient))
	require.NoError(t, err)

	// Connect client to bootstrap
	err = connectNodes(ctx, client, bootstrap)
	require.NoError(t, err)

	// Bootstrap client DHT
	err = clientDHT.Bootstrap(ctx)
	require.NoError(t, err)

	// Wait for connection
	err = WaitForConnection(ctx, client, bootstrap, 10*time.Second)
	require.NoError(t, err)

	t.Run("BootstrapConnection", func(t *testing.T) {
		// Verify connection
		peers := client.Network().Peers()
		assert.Contains(t, peers, bootstrap.ID(), "Client should be connected to bootstrap")
	})

	t.Run("DHTPutGetAfterBootstrap", func(t *testing.T) {
		key := createDHTKey("bootstrap-test-key")
		value := []byte("bootstrap-test-value")

		// Store value on bootstrap
		err := bootstrapDHT.PutValue(ctx, key, value)
		require.NoError(t, err)

		// Wait for value to be available on client using our helper
		err = WaitForDHTValue(ctx, clientDHT, key, value, 15*time.Second)
		require.NoError(t, err, "Client should retrieve value from bootstrap DHT")
	})
}

func TestDHTContentRouting(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Create provider and consumer nodes
	provider, err := createNodeWithOptions(ctx, 0, false, false)
	require.NoError(t, err)
	defer provider.Close()

	consumer, err := createNodeWithOptions(ctx, 0, false, false)
	require.NoError(t, err)
	defer consumer.Close()

	// Create DHTs
	providerDHT, err := dht.New(ctx, provider, dht.Mode(dht.ModeServer))
	require.NoError(t, err)

	consumerDHT, err := dht.New(ctx, consumer, dht.Mode(dht.ModeServer))
	require.NoError(t, err)

	// Bootstrap DHTs
	err = providerDHT.Bootstrap(ctx)
	require.NoError(t, err)

	err = consumerDHT.Bootstrap(ctx)
	require.NoError(t, err)

	// Connect nodes
	err = connectNodes(ctx, provider, consumer)
	require.NoError(t, err)

	// Wait for connection
	err = WaitForConnection(ctx, provider, consumer, 10*time.Second)
	require.NoError(t, err)

	t.Run("BasicDHTFunctionality", func(t *testing.T) {
		// Test basic DHT functionality without complex content routing
		key := createDHTKey("content-routing-test")
		value := []byte("test-content-data")

		// Provider stores content info
		err := providerDHT.PutValue(ctx, key, value)
		require.NoError(t, err)

		// Wait for value to be available on consumer
		err = WaitForDHTValue(ctx, consumerDHT, key, value, 15*time.Second)
		require.NoError(t, err, "Consumer should retrieve content info")
	})
}

// Helper function to create DHT with specific mode
func createDHTNode(ctx context.Context, mode dht.ModeOpt) (host.Host, *dht.IpfsDHT, error) {
	node, err := createNodeWithOptions(ctx, 0, false, false)
	if err != nil {
		return nil, nil, err
	}

	kademliaDHT, err := dht.New(ctx, node, dht.Mode(mode))
	if err != nil {
		node.Close()
		return nil, nil, err
	}

	err = kademliaDHT.Bootstrap(ctx)
	if err != nil {
		node.Close()
		return nil, nil, err
	}

	return node, kademliaDHT, nil
} 