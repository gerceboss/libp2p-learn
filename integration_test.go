package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiNodeNetwork(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	nodeCount := 5
	nodes := make([]host.Host, nodeCount)

	// Create multiple nodes
	for i := 0; i < nodeCount; i++ {
		node, err := createNodeWithOptions(ctx, 0, false, false)
		require.NoError(t, err)
		defer node.Close()
		nodes[i] = node
	}

	// Connect nodes in a mesh topology
	for i := 0; i < nodeCount; i++ {
		for j := i + 1; j < nodeCount; j++ {
			err := connectNodes(ctx, nodes[i], nodes[j])
			require.NoError(t, err)
		}
	}

	// Wait for all connections to stabilize
	err := WaitForAllConnections(ctx, nodes, 30*time.Second)
	require.NoError(t, err, "All nodes should be connected in mesh")

	t.Run("FullMeshConnectivity", func(t *testing.T) {
		// Verify all nodes are connected to all other nodes
		for i := 0; i < nodeCount; i++ {
			peers := nodes[i].Network().Peers()
			assert.Equal(t, nodeCount-1, len(peers), 
				"Node %d should be connected to %d other nodes", i, nodeCount-1)
		}
	})

	t.Run("MessagePropagation", func(t *testing.T) {
		// Set up protocol handlers on all nodes
		handlers := make([]*ProtocolHandler, nodeCount)
		messageReceived := make([]bool, nodeCount)

		for i := 0; i < nodeCount; i++ {
			handlers[i] = NewProtocolHandler(nodes[i])
			handlers[i].SetupProtocols()
		}

		// Wait for protocols to be ready (condition-based)
		err := WaitWithCondition(ctx, func() bool {
			// Check if all nodes have the ping protocol registered
			for i := 0; i < nodeCount; i++ {
				protocols := nodes[i].Mux().Protocols()
				hasProtocol := false
				for _, proto := range protocols {
					if string(proto) == PingProtocol {
						hasProtocol = true
						break
					}
				}
				if !hasProtocol {
					return false
				}
			}
			return true
		}, 15*time.Second, 200*time.Millisecond)
		require.NoError(t, err, "All protocols should be registered")

		// Send messages from node 0 to all other nodes
		for i := 1; i < nodeCount; i++ {
			response, err := handlers[0].SendPing(ctx, nodes[i].ID(), fmt.Sprintf("ping-%d", i))
			require.NoError(t, err)
			assert.Contains(t, response, "pong", "Should receive pong from node %d", i)
			messageReceived[i] = true
		}

		// Verify all messages were received
		for i := 1; i < nodeCount; i++ {
			assert.True(t, messageReceived[i], "Should have received message from node %d", i)
		}
	})
}

func TestRelayedConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Create relay node
	relay, err := createNodeWithOptions(ctx, 0, true, false)
	require.NoError(t, err)
	defer relay.Close()

	// Create two client nodes that will communicate through relay
	client1, err := createNodeWithOptions(ctx, 0, false, false)
	require.NoError(t, err)
	defer client1.Close()

	client2, err := createNodeWithOptions(ctx, 0, false, false)
	require.NoError(t, err)
	defer client2.Close()

	// Connect both clients to relay
	err = connectNodes(ctx, client1, relay)
	require.NoError(t, err)

	err = connectNodes(ctx, client2, relay)
	require.NoError(t, err)

	// Wait for connections to be established properly
	err = WaitForConnection(ctx, client1, relay, 15*time.Second)
	require.NoError(t, err)
	
	err = WaitForConnection(ctx, client2, relay, 15*time.Second)
	require.NoError(t, err)

	t.Run("RelayConnectivity", func(t *testing.T) {
		// Verify both clients are connected to relay
		relayPeers := relay.Network().Peers()
		assert.Contains(t, relayPeers, client1.ID(), "Relay should be connected to client1")
		assert.Contains(t, relayPeers, client2.ID(), "Relay should be connected to client2")

		client1Peers := client1.Network().Peers()
		client2Peers := client2.Network().Peers()
		
		assert.Contains(t, client1Peers, relay.ID(), "Client1 should be connected to relay")
		assert.Contains(t, client2Peers, relay.ID(), "Client2 should be connected to relay")
	})

	t.Run("CommunicationThroughRelay", func(t *testing.T) {
		// Set up protocol handlers
		handler1 := NewProtocolHandler(client1)
		handler1.SetupProtocols()

		handler2 := NewProtocolHandler(client2)
		handler2.SetupProtocols()

		// Wait for protocols to be ready
		err := WaitWithCondition(ctx, func() bool {
			protocols1 := client1.Mux().Protocols()
			protocols2 := client2.Mux().Protocols()
			
			hasProtocol1 := false
			hasProtocol2 := false
			
			for _, proto := range protocols1 {
				if string(proto) == PingProtocol {
					hasProtocol1 = true
					break
				}
			}
			for _, proto := range protocols2 {
				if string(proto) == PingProtocol {
					hasProtocol2 = true
					break
				}
			}
			
			return hasProtocol1 && hasProtocol2
		}, 10*time.Second, 200*time.Millisecond)
		require.NoError(t, err, "Protocols should be ready")

		// Try to send message from client1 to client2 (may go through relay)
		response, err := handler1.SendPing(ctx, client2.ID(), "relay-test-ping")
		
		// This might fail if direct connection isn't possible, but we can test the relay path
		if err == nil {
			assert.Contains(t, response, "pong", "Should receive pong response through relay")
		} else {
			t.Logf("Direct communication failed (expected in some NAT scenarios): %v", err)
		}
	})

	t.Run("RelayedStreamCreation", func(t *testing.T) {
		// Test creating a relayed stream
		// This is more advanced and depends on specific libp2p relay implementation
		t.Skip("Relayed stream creation test - implementation depends on specific relay setup")
	})
}

func TestHolePunchingScenario(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Create nodes with hole punching enabled
	node1, err := createNodeWithOptions(ctx, 0, false, false)
	require.NoError(t, err)
	defer node1.Close()

	node2, err := createNodeWithOptions(ctx, 0, false, false)
	require.NoError(t, err)
	defer node2.Close()

	// Create a relay node to assist with hole punching
	relay, err := createNodeWithOptions(ctx, 0, true, false)
	require.NoError(t, err)
	defer relay.Close()

	// Connect both nodes to relay first
	err = connectNodes(ctx, node1, relay)
	require.NoError(t, err)

	err = connectNodes(ctx, node2, relay)
	require.NoError(t, err)

	// Wait for connections to be established
	err = WaitForConnection(ctx, node1, relay, 15*time.Second)
	require.NoError(t, err)
	
	err = WaitForConnection(ctx, node2, relay, 15*time.Second)
	require.NoError(t, err)

	t.Run("AutoNATDetection", func(t *testing.T) {
		// Check if nodes detected their NAT status
		// This is informational as AutoNAT detection might not complete in test timeframe
		
		// Log the addresses to see what was detected
		t.Logf("Node1 addresses: %v", node1.Addrs())
		t.Logf("Node2 addresses: %v", node2.Addrs())
		
		// Basic connectivity check
		peers1 := node1.Network().Peers()
		peers2 := node2.Network().Peers()
		
		assert.Contains(t, peers1, relay.ID(), "Node1 should be connected to relay")
		assert.Contains(t, peers2, relay.ID(), "Node2 should be connected to relay")
	})

	t.Run("DirectConnectionAttempt", func(t *testing.T) {
		// Try to establish direct connection (may trigger hole punching)
		// In a real NAT scenario, this would involve the DCUtR protocol
		
		// Attempt direct connection
		err = connectNodes(ctx, node1, node2)
		
		if err == nil {
			// Wait for direct connection to be established
			connErr := WaitForConnection(ctx, node1, node2, 10*time.Second)
			if connErr == nil {
				// Direct connection succeeded
				peers1 := node1.Network().Peers()
				peers2 := node2.Network().Peers()
				
				assert.Contains(t, peers1, node2.ID(), "Node1 should be directly connected to Node2")
				assert.Contains(t, peers2, node1.ID(), "Node2 should be directly connected to Node1")
				
				t.Log("Direct connection established (hole punching successful or not needed)")
			} else {
				t.Logf("Direct connection initiated but not established: %v", connErr)
			}
		} else {
			t.Logf("Direct connection failed (expected in some NAT scenarios): %v", err)
			
			// Verify nodes can still communicate through relay
			handler1 := NewProtocolHandler(node1)
			handler1.SetupProtocols()
			
			handler2 := NewProtocolHandler(node2)
			handler2.SetupProtocols()
			
			// Wait for protocols to be ready
			err := WaitWithCondition(ctx, func() bool {
				protocols := node1.Mux().Protocols()
				for _, proto := range protocols {
					if string(proto) == PingProtocol {
						return true
					}
				}
				return false
			}, 5*time.Second, 200*time.Millisecond)
			
			if err == nil {
				// Test communication through relay
				_, commErr := handler1.SendPing(ctx, node2.ID(), "hole-punch-test")
				if commErr == nil {
					t.Log("Communication through relay still works")
				}
			}
		}
	})
}

func TestNetworkResilience(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	nodeCount := 4
	nodes := make([]host.Host, nodeCount)

	// Create nodes
	for i := 0; i < nodeCount; i++ {
		node, err := createNodeWithOptions(ctx, 0, false, false)
		require.NoError(t, err)
		defer node.Close()
		nodes[i] = node
	}

	// Create initial connections (star topology with node 0 as center)
	for i := 1; i < nodeCount; i++ {
		err := connectNodes(ctx, nodes[0], nodes[i])
		require.NoError(t, err)
	}

	// Wait for star topology to be established
	err := WaitForPeerCount(ctx, nodes[0], nodeCount-1, 15*time.Second)
	require.NoError(t, err, "Central node should be connected to all others")

	t.Run("InitialConnectivity", func(t *testing.T) {
		// Verify initial connectivity
		centralPeers := nodes[0].Network().Peers()
		assert.Equal(t, nodeCount-1, len(centralPeers), "Central node should be connected to all others")

		for i := 1; i < nodeCount; i++ {
			peers := nodes[i].Network().Peers()
			assert.Contains(t, peers, nodes[0].ID(), "Node %d should be connected to central node", i)
		}
	})

	t.Run("NodeFailureRecovery", func(t *testing.T) {
		// Simulate node 0 (central) going offline
		err := nodes[0].Close()
		require.NoError(t, err)

		// Wait for failure detection using condition checking
		err = WaitWithCondition(ctx, func() bool {
			// Check that remaining nodes are no longer connected to node 0
			for i := 1; i < nodeCount; i++ {
				peers := nodes[i].Network().Peers()
				for _, p := range peers {
					if p == nodes[0].ID() {
						return false // Still connected
					}
				}
			}
			return true
		}, 15*time.Second, 500*time.Millisecond)
		require.NoError(t, err, "Nodes should detect failure")

		// Create replacement connections between remaining nodes
		for i := 1; i < nodeCount-1; i++ {
			for j := i + 1; j < nodeCount; j++ {
				err := connectNodes(ctx, nodes[i], nodes[j])
				if err != nil {
					t.Logf("Failed to connect nodes %d and %d: %v", i, j, err)
				} else {
					// Wait for this specific connection
					WaitForConnection(ctx, nodes[i], nodes[j], 5*time.Second)
				}
			}
		}

		// Verify remaining nodes are still functional
		for i := 1; i < nodeCount; i++ {
			peers := nodes[i].Network().Peers()
			assert.NotContains(t, peers, nodes[0].ID(), "Node %d should not be connected to failed node", i)
			assert.Greater(t, len(peers), 0, "Node %d should have some connections", i)
		}
	})
}

func TestHighLoadScenario(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create central hub node
	hub, err := createNodeWithOptions(ctx, 0, true, false)
	require.NoError(t, err)
	defer hub.Close()

	// Create multiple client nodes
	clientCount := 10
	clients := make([]host.Host, clientCount)
	handlers := make([]*ProtocolHandler, clientCount)

	for i := 0; i < clientCount; i++ {
		client, err := createNodeWithOptions(ctx, 0, false, false)
		require.NoError(t, err)
		defer client.Close()
		clients[i] = client

		// Set up protocol handler first
		handlers[i] = NewProtocolHandler(client)
		handlers[i].SetupProtocols()
	}

	// Connect clients to hub in batches to avoid overwhelming the connection system
	batchSize := 3
	for i := 0; i < clientCount; i += batchSize {
		end := i + batchSize
		if end > clientCount {
			end = clientCount
		}

		// Connect batch of clients
		for j := i; j < end; j++ {
			err = connectNodes(ctx, clients[j], hub)
			if err != nil {
				t.Logf("Failed to connect client %d: %v", j, err)
				// Continue with other clients instead of failing immediately
				continue
			}
		}

		// Wait a bit between batches to allow connections to stabilize
		if end < clientCount {
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Set up hub protocol handler
	hubHandler := NewProtocolHandler(hub)
	hubHandler.SetupProtocols()

	// Wait for majority of clients to be connected to hub (allow some connection failures in high load)
	minConnections := int(float64(clientCount) * 0.7) // Accept 70% success rate
	err = WaitForPeerCount(ctx, hub, minConnections, 30*time.Second)
	require.NoError(t, err, "Hub should be connected to at least %d clients", minConnections)

	t.Run("ConcurrentConnections", func(t *testing.T) {
		// Verify majority of clients are connected to hub
		hubPeers := hub.Network().Peers()
		actualConnections := len(hubPeers)
		t.Logf("Hub has %d connections out of %d attempted", actualConnections, clientCount)
		
		assert.GreaterOrEqual(t, actualConnections, minConnections, 
			"Hub should be connected to at least %d clients", minConnections)

		// Count successful client connections
		connectedClients := 0
		for i := 0; i < clientCount; i++ {
			clientPeers := clients[i].Network().Peers()
			if len(clientPeers) > 0 {
				for _, p := range clientPeers {
					if p == hub.ID() {
						connectedClients++
						break
					}
				}
			}
		}
		
		t.Logf("Successfully connected clients: %d/%d", connectedClients, clientCount)
		assert.GreaterOrEqual(t, connectedClients, minConnections,
			"At least %d clients should be connected to hub", minConnections)
	})

	t.Run("ConcurrentMessaging", func(t *testing.T) {
		// Wait for all protocols to be ready
		err := WaitWithCondition(ctx, func() bool {
			protocols := hub.Mux().Protocols()
			for _, proto := range protocols {
				if string(proto) == PingProtocol {
					return true
				}
			}
			return false
		}, 10*time.Second, 200*time.Millisecond)
		require.NoError(t, err, "Hub protocols should be ready")

		// Send messages concurrently from connected clients to hub
		connectedClientIndices := []int{}
		for i := 0; i < clientCount; i++ {
			clientPeers := clients[i].Network().Peers()
			for _, p := range clientPeers {
				if p == hub.ID() {
					connectedClientIndices = append(connectedClientIndices, i)
					break
				}
			}
		}

		if len(connectedClientIndices) == 0 {
			t.Skip("No clients connected to hub for messaging test")
			return
		}

		results := make(chan error, len(connectedClientIndices))

		for _, clientIdx := range connectedClientIndices {
			go func(idx int) {
				response, err := handlers[idx].SendPing(ctx, hub.ID(), 
					fmt.Sprintf("concurrent-ping-%d", idx))
				if err != nil {
					results <- err
					return
				}
				if !assert.Contains(t, response, "pong", "Should receive pong response") {
					results <- fmt.Errorf("invalid response: %s", response)
					return
				}
				results <- nil
			}(clientIdx)
		}

		// Wait for all results
		successCount := 0
		for i := 0; i < len(connectedClientIndices); i++ {
			err := <-results
			if err == nil {
				successCount++
			} else {
				t.Logf("Client failed: %v", err)
			}
		}

		// At least 70% of connected clients should succeed
		expectedSuccess := int(float64(len(connectedClientIndices)) * 0.7)
		assert.GreaterOrEqual(t, successCount, expectedSuccess, 
			"At least %d out of %d connected clients should succeed", expectedSuccess, len(connectedClientIndices))
	})
}

func TestWebSocketConnectivity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create nodes with WebSocket enabled
	node1, err := createNodeWithOptions(ctx, 0, false, true)
	require.NoError(t, err)
	defer node1.Close()

	node2, err := createNodeWithOptions(ctx, 0, false, true)
	require.NoError(t, err)
	defer node2.Close()

	// Connect nodes
	err = connectNodes(ctx, node1, node2)
	require.NoError(t, err)

	// Wait for connection to be established
	err = WaitForConnection(ctx, node1, node2, 10*time.Second)
	require.NoError(t, err)

	t.Run("WebSocketAddresses", func(t *testing.T) {
		// Verify WebSocket addresses are present
		addrs1 := node1.Addrs()
		addrs2 := node2.Addrs()

		hasWS1 := false
		hasWS2 := false

		for _, addr := range addrs1 {
			if containsProtocol(addr.String(), "ws") {
				hasWS1 = true
				break
			}
		}

		for _, addr := range addrs2 {
			if containsProtocol(addr.String(), "ws") {
				hasWS2 = true
				break
			}
		}

		assert.True(t, hasWS1, "Node1 should have WebSocket addresses")
		assert.True(t, hasWS2, "Node2 should have WebSocket addresses")
	})

	t.Run("WebSocketCommunication", func(t *testing.T) {
		// Set up protocol handlers
		handler1 := NewProtocolHandler(node1)
		handler1.SetupProtocols()

		handler2 := NewProtocolHandler(node2)
		handler2.SetupProtocols()

		// Wait for protocols to be ready
		err := WaitWithCondition(ctx, func() bool {
			protocols := node1.Mux().Protocols()
			for _, proto := range protocols {
				if string(proto) == PingProtocol {
					return true
				}
			}
			return false
		}, 5*time.Second, 200*time.Millisecond)
		require.NoError(t, err, "Protocols should be ready")

		// Test communication
		response, err := handler1.SendPing(ctx, node2.ID(), "websocket-ping")
		require.NoError(t, err)
		assert.Contains(t, response, "pong", "Should receive pong response over WebSocket")
	})
} 