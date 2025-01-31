package p2p

import (
	"context"
	"fmt"
	"goker/internal/channelmanager"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/multiformats/go-multiaddr"
)

// 'Host' handler

// Notification system for host - Handles incoming connections and disconnections
func (p *GokerPeer) handleNotifications() {
	p.thisHost.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, conn network.Conn) { // On peer connect
			fmt.Printf("NOTIFICATION: Connection from new peer: %s\n", conn.RemotePeer()) // WHEN A NEW PEER CONNECTS TO US, IT MUST BE FROM A BROADCAST SERVER SENDING IT

			// Connect to the new peer and update the peer list
			p.handlePeerConnection(conn.RemotePeer(), conn.RemoteMultiaddr())
			channelmanager.FNET_NumOfPlayersChan <- len(p.peerList)
		},
		DisconnectedF: func(n network.Network, conn network.Conn) { // On peer disconnect
			fmt.Printf("NOTIFICATION: Disconnected from peer: %s\n", conn.RemotePeer())

			p.handlePeerDisconnection(conn.RemotePeer())
			channelmanager.FNET_NumOfPlayersChan <- len(p.peerList)
		},
	})

	// Run this function forever - IF YOU REMOVE THIS, THE PROGRAM WILL CLOSE
	select {}
}

// Connect to new peers that are discovered - Called when the ConnectF NOTIFICATION has been made.
func (p *GokerPeer) handlePeerConnection(newPeerID peer.ID, newPeerAddr multiaddr.Multiaddr) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	// Check if already connected
	if _, exists := p.peerList[newPeerID]; exists {
		log.Printf("Already connected to peer: %s\n", newPeerID)
		return
	}

	// Add new peer to the peer list
	p.peerList[newPeerID] = newPeerAddr

	// Create address info for the new peer
	addrInfo := peer.AddrInfo{ID: newPeerID, Addrs: []multiaddr.Multiaddr{newPeerAddr}}

	// Attempt to connect to the new peer
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.thisHost.Connect(ctx, addrInfo); err != nil {
		log.Printf("Failed to connect to new peer %s: %v", newPeerID, err)
		return
	}
	fmt.Printf("Connected to new peer: %s\n", newPeerID)

	// Request Nickname from new peer
	p.ExecuteCommand(&NicknameRequestCommand{})
}

// Handle existing peer disconnection - Called when the DisconnectF NOTIFICATION has been made
func (p *GokerPeer) handlePeerDisconnection(peerID peer.ID) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	// Remove the peer from neccessary maps
	if _, exists := p.peerList[peerID]; exists {
		delete(p.peerList, peerID)
		delete(p.peerNicknames, peerID)
		fmt.Printf("Peer %s disconnected and removed from peer list.\n", peerID)
	} else {
		log.Printf("Peer %s not found in peer list.\n", peerID)
	}
}
