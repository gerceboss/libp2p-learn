package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/sirupsen/logrus"
)

const (
	// Protocol IDs
	PingProtocol = "/libp2p-learn/ping/1.0.0"
	ChatProtocol = "/libp2p-learn/chat/1.0.0"
	EchoProtocol = "/libp2p-learn/echo/1.0.0"
)

// ProtocolHandler manages custom protocols for the node
type ProtocolHandler struct {
	host host.Host
}

// NewProtocolHandler creates a new protocol handler
func NewProtocolHandler(h host.Host) *ProtocolHandler {
	return &ProtocolHandler{host: h}
}

// SetupProtocols registers all custom protocols
func (p *ProtocolHandler) SetupProtocols() {
	// Register ping protocol
	p.host.SetStreamHandler(protocol.ID(PingProtocol), p.handlePing)
	logrus.WithField("protocol", PingProtocol).Info("Registered ping protocol")

	// Register chat protocol
	p.host.SetStreamHandler(protocol.ID(ChatProtocol), p.handleChat)
	logrus.WithField("protocol", ChatProtocol).Info("Registered chat protocol")

	// Register echo protocol
	p.host.SetStreamHandler(protocol.ID(EchoProtocol), p.handleEcho)
	logrus.WithField("protocol", EchoProtocol).Info("Registered echo protocol")
}

// handlePing handles incoming ping requests
func (p *ProtocolHandler) handlePing(s network.Stream) {
	defer s.Close()

	peer := s.Conn().RemotePeer()
	logrus.WithField("peer", peer).Debug("Received ping request")

	// Read the ping message
	reader := bufio.NewReader(s)
	data, err := reader.ReadString('\n')
	if err != nil {
		logrus.WithError(err).Error("Failed to read ping data")
		return
	}

	// Send pong response
	writer := bufio.NewWriter(s)
	_, err = writer.WriteString(fmt.Sprintf("pong: %s", data))
	if err != nil {
		logrus.WithError(err).Error("Failed to write pong response")
		return
	}
	writer.Flush()

	logrus.WithFields(logrus.Fields{
		"peer": peer,
		"data": data[:len(data)-1], // Remove newline
	}).Info("Handled ping request")
}

// handleChat handles incoming chat messages
func (p *ProtocolHandler) handleChat(s network.Stream) {
	defer s.Close()

	peer := s.Conn().RemotePeer()
	logrus.WithField("peer", peer).Debug("Received chat connection")

	reader := bufio.NewReader(s)
	writer := bufio.NewWriter(s)

	for {
		// Read message
		message, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				logrus.WithError(err).Error("Failed to read chat message")
			}
			break
		}

		logrus.WithFields(logrus.Fields{
			"peer":    peer,
			"message": message[:len(message)-1], // Remove newline
		}).Info("Received chat message")

		// Echo the message back with timestamp
		response := fmt.Sprintf("[%s] Echo: %s", time.Now().Format("15:04:05"), message)
		_, err = writer.WriteString(response)
		if err != nil {
			logrus.WithError(err).Error("Failed to write chat response")
			break
		}
		writer.Flush()
	}
}

// handleEcho handles incoming echo requests
func (p *ProtocolHandler) handleEcho(s network.Stream) {
	defer s.Close()

	peer := s.Conn().RemotePeer()
	logrus.WithField("peer", peer).Debug("Received echo connection")

	// Simple echo - copy input to output
	_, err := io.Copy(s, s)
	if err != nil {
		logrus.WithError(err).Error("Failed to echo data")
		return
	}

	logrus.WithField("peer", peer).Info("Handled echo request")
}

// SendPing sends a ping to a peer
func (p *ProtocolHandler) SendPing(ctx context.Context, peerID peer.ID, message string) (string, error) {
	s, err := p.host.NewStream(ctx, peerID, protocol.ID(PingProtocol))
	if err != nil {
		return "", fmt.Errorf("failed to create stream: %w", err)
	}
	defer s.Close()

	// Send ping
	writer := bufio.NewWriter(s)
	_, err = writer.WriteString(message + "\n")
	if err != nil {
		return "", fmt.Errorf("failed to send ping: %w", err)
	}
	writer.Flush()

	// Read pong
	reader := bufio.NewReader(s)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read pong: %w", err)
	}

	return response[:len(response)-1], nil // Remove newline
}

// SendChatMessage sends a chat message to a peer
func (p *ProtocolHandler) SendChatMessage(ctx context.Context, peerID peer.ID, message string) (string, error) {
	s, err := p.host.NewStream(ctx, peerID, protocol.ID(ChatProtocol))
	if err != nil {
		return "", fmt.Errorf("failed to create stream: %w", err)
	}
	defer s.Close()

	writer := bufio.NewWriter(s)
	reader := bufio.NewReader(s)

	// Send message
	_, err = writer.WriteString(message + "\n")
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}
	writer.Flush()

	// Read response
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return response[:len(response)-1], nil // Remove newline
}

// SendEcho sends data to echo protocol
func (p *ProtocolHandler) SendEcho(ctx context.Context, peerID peer.ID, data string) (string, error) {
	s, err := p.host.NewStream(ctx, peerID, protocol.ID(EchoProtocol))
	if err != nil {
		return "", fmt.Errorf("failed to create stream: %w", err)
	}
	defer s.Close()

	// Send data
	_, err = s.Write([]byte(data))
	if err != nil {
		return "", fmt.Errorf("failed to send data: %w", err)
	}

	// Close write side to signal EOF
	s.CloseWrite()

	// Read echoed data
	response, err := io.ReadAll(s)
	if err != nil {
		return "", fmt.Errorf("failed to read echo: %w", err)
	}

	return string(response), nil
}
