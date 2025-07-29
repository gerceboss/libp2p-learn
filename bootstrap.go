package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

// bootstrapPeers connects to the given bootstrap peers
func bootstrapPeers(ctx context.Context, h host.Host, peers []string) error {
	if len(peers) == 0 {
		return nil
	}

	logrus.WithField("count", len(peers)).Info("Starting bootstrap process")

	var wg sync.WaitGroup
	for _, peerAddr := range peers {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			if err := connectToPeer(ctx, h, addr); err != nil {
				logrus.WithError(err).WithField("peer", addr).Error("Failed to connect to bootstrap peer")
			}
		}(peerAddr)
	}

	wg.Wait()
	logrus.Info("Bootstrap process completed")
	return nil
}

// connectToPeer connects to a single peer
func connectToPeer(ctx context.Context, h host.Host, peerAddr string) error {
	addr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr %s: %w", peerAddr, err)
	}

	peerinfo, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return fmt.Errorf("failed to get peer info from %s: %w", peerAddr, err)
	}

	if err := h.Connect(ctx, *peerinfo); err != nil {
		return fmt.Errorf("failed to connect to %s: %w", peerinfo.ID, err)
	}

	logrus.WithField("peer", peerinfo.ID).Info("Successfully connected to peer")
	return nil
}

// getConnectedPeers returns information about currently connected peers
func getConnectedPeers(h host.Host) []peer.ID {
	return h.Network().Peers()
}

// printPeerInfo displays information about connected peers
func printPeerInfo(h host.Host) {
	peers := getConnectedPeers(h)
	logrus.WithField("count", len(peers)).Info("Connected peers")

	for i, p := range peers {
		conns := h.Network().ConnsToPeer(p)
		if len(conns) > 0 {
			protocols, err := h.Peerstore().GetProtocols(p)
			if err != nil {
				logrus.WithError(err).WithField("peer", p).Error("Failed to get protocols for peer")
				continue
			}
			
			logrus.WithFields(logrus.Fields{
				"index":     i + 1,
				"peer_id":   p,
				"addresses": h.Peerstore().Addrs(p),
				"protocols": protocols,
			}).Info("Peer info")
		}
	}
}
