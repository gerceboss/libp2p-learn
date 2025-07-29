package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "libp2p-node",
		Short: "A libp2p node with TCP/UDP/WebSocket support and hole punching",
		Run:   runNode,
	}

	var port int
	var enableRelay bool
	var bootstrap []string
	var configFile string
	var enableWebSocket bool

	rootCmd.Flags().IntVarP(&port, "port", "p", 0, "Port to listen on (0 for random)")
	rootCmd.Flags().BoolVarP(&enableRelay, "relay", "r", false, "Enable relay functionality")
	rootCmd.Flags().StringArrayVarP(&bootstrap, "bootstrap", "b", nil, "Bootstrap peer addresses")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file path")
	rootCmd.Flags().BoolVarP(&enableWebSocket, "websocket", "w", true, "Enable WebSocket transport")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runNode(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	configFile, _ := cmd.Flags().GetString("config")
	config, err := LoadConfig(configFile)
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Override config with CLI flags
	if port, _ := cmd.Flags().GetInt("port"); port != 0 {
		config.ListenPort = port
	}
	if enableRelay, _ := cmd.Flags().GetBool("relay"); enableRelay {
		config.EnableRelay = true
	}
	if bootstrap, _ := cmd.Flags().GetStringArray("bootstrap"); len(bootstrap) > 0 {
		config.BootstrapPeers = bootstrap
	}
	if enableWebSocket, _ := cmd.Flags().GetBool("websocket"); !enableWebSocket {
		config.EnableWebSocket = false
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		log.Fatal("Invalid configuration:", err)
	}

	// Setup logging
	if err := config.SetupLogging(); err != nil {
		log.Fatal("Failed to setup logging:", err)
	}

	fmt.Printf("Starting libp2p node...\n")
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Port: %d\n", config.ListenPort)
	fmt.Printf("  Enable Relay: %t\n", config.EnableRelay)
	fmt.Printf("  Enable Hole Punching: %t\n", config.EnableHolePunch)
	fmt.Printf("  Enable WebSocket: %t\n", config.EnableWebSocket)
	fmt.Printf("  Max Connections: %d\n", config.MaxConnections)
	fmt.Printf("  Bootstrap Peers: %d\n", len(config.BootstrapPeers))

	// Create the libp2p node
	fmt.Println("Creating libp2p node...")
	node, err := createNodeWithOptions(ctx, config.ListenPort, config.EnableRelay, config.EnableWebSocket)
	if err != nil {
		log.Fatal("Failed to create node:", err)
	}
	defer node.Close()

	fmt.Printf("Node started successfully!\n")
	fmt.Printf("Node ID: %s\n", node.ID())
	fmt.Printf("Listening addresses:\n")
	for _, addr := range node.Addrs() {
		fmt.Printf("  %s/p2p/%s\n", addr, node.ID())
	}

	// Set up protocols
	protocolHandler := NewProtocolHandler(node)
	protocolHandler.SetupProtocols()

	// Bootstrap process
	if len(config.BootstrapPeers) > 0 {
		fmt.Printf("Bootstrapping with %d peers...\n", len(config.BootstrapPeers))
		if err := bootstrapPeers(ctx, node, config.BootstrapPeers); err != nil {
			log.Printf("Bootstrap error: %v", err)
		}
	}

	fmt.Println("\nNode is running. Features enabled:")
	fmt.Printf("  ✓ TCP Transport\n")
	fmt.Printf("  ✓ UDP/QUIC Transport\n")
	if config.EnableWebSocket {
		fmt.Printf("  ✓ WebSocket/WSS Transport\n")
	}
	fmt.Printf("  ✓ Connection Management (max: %d)\n", config.MaxConnections)
	if config.EnableHolePunch {
		fmt.Printf("  ✓ Hole Punching/NAT Traversal\n")
	}
	if config.EnableRelay {
		fmt.Printf("  ✓ Relay Service\n")
	}
	if config.EnableAutoNAT {
		fmt.Printf("  ✓ AutoNAT\n")
	}

	// Show peer info periodically
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				printPeerInfo(node)
			}
		}
	}()

	fmt.Println("\nPress Ctrl+C to stop...")

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\nShutting down...")
	time.Sleep(500 * time.Millisecond)
	fmt.Println("Node stopped")
}
