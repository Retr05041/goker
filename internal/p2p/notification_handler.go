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
	p.ThisHost.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, conn network.Conn) { // On peer connect
			fmt.Printf("NOTIFICATION: Connection from new peer: %s\n", conn.RemotePeer()) // WHEN A NEW PEER CONNECTS TO US, IT MUST BE FROM A BROADCAST SERVER SENDING IT

			// Connect to the new peer and update the peer list
			err := p.handlePeerConnection(conn.RemotePeer(), conn.RemoteMultiaddr())
			if err != nil {
				log.Println("handlePeerConnection failed: ", err)
				return
			}

			// Update GUI
			channelmanager.FNET_NumOfPlayersChan <- len(p.peerList)

			// Exchange keys
			p.ExecuteCommand(&PubKeyExchangeCommand{})

			// Request Nickname from new peer
			p.ExecuteCommand(&NicknameRequestCommand{})
		},
		DisconnectedF: func(n network.Network, conn network.Conn) { // On peer disconnect
			fmt.Printf("NOTIFICATION: Disconnected from peer: %s\n", conn.RemotePeer())

			if !p.gameState.FoldedPlayers[conn.RemotePeer()] { // If the person who left hasn't folded
				p.gameState.SomeoneLeft = true
			}

			// Update the peers list and nicknames
			p.handlePeerDisconnection(conn.RemotePeer())

			// Update the GUI
			channelmanager.FNET_NumOfPlayersChan <- len(p.peerList)

			// Remove the peer from the state
			p.gameState.RemovePeerFromState(conn.RemotePeer())

			// Update GUI of player leaving
			channelmanager.TGUI_PlayerInfo <- p.gameState.GetPlayerInfo()

			if p.gameState.TurnOrder[p.gameState.WhosTurn] == conn.RemotePeer() {
				p.gameState.NextTurn()
			}
		},
	})

	// Run this function forever - IF YOU REMOVE THIS, THE PROGRAM WILL CLOSE
	select {}
}

// Connect to new peers that are discovered - Called when the ConnectF NOTIFICATION has been made.
func (p *GokerPeer) handlePeerConnection(newPeerID peer.ID, newPeerAddr multiaddr.Multiaddr) error {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	// Add new peer to the peer list
	p.peerList = append(p.peerList, peerInfo{ID: newPeerID, Addr: newPeerAddr})

	// Create address info for the new peer
	addrInfo := peer.AddrInfo{ID: newPeerID, Addrs: []multiaddr.Multiaddr{newPeerAddr}}

	// Attempt to connect to the new peer
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.ThisHost.Connect(ctx, addrInfo); err != nil {
		return fmt.Errorf("failed to connect to new peer %s: %v", newPeerID, err)
	}
	fmt.Printf("Connected to new peer: %s\n", newPeerID)
	return nil
}

// Handle existing peer disconnection - Called when the DisconnectF NOTIFICATION has been made
func (p *GokerPeer) handlePeerDisconnection(peerID peer.ID) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	for i, peerInfo := range p.peerList {
		if peerInfo.ID == peerID {
			p.peerList = append(p.peerList[:i], p.peerList[i+1:]...)
		}
	}
}
