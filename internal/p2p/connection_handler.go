package p2p

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/multiformats/go-multiaddr"
)

var protocolID = protocol.ID("/goker/1.0.0")

// Handle new peers being discovered (connected to this host, and disconnected from this host)
func (p *GokerPeer) handleNotifications() {
	p.thisHost.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, conn network.Conn) { // On peer connect
			fmt.Printf("NOTIFICATION: Connection from new peer: %s\n", conn.RemotePeer()) // WHEN A NEW PEER CONNECTS TO US, IT MUST BE FROM A BROADCAST SERVER SENDING IT

			// Connect to the new peer and update the peer list
			p.handlePeerConnection(conn.RemotePeer(), conn.RemoteMultiaddr())
		},
		DisconnectedF: func(n network.Network, conn network.Conn) { // On peer disconnect
			fmt.Printf("NOTIFICATION: Disconnected from peer: %s\n", conn.RemotePeer())

			p.handlePeerDisconnection(conn.RemotePeer())
		},
	})

	// Run this function forever - IF YOU REMOVE THIS, THE PROGRAM WILL CLOSE
	select {}
}

// Log connected peers - Ignore self - For debugging only
func (p *GokerPeer) alert() {
	fmt.Println("LOGGING HAS BEGUN")
	for {
		p.peerListMutex.Lock()
		if len(p.peerList) > 1 { // Account for this host
			fmt.Printf("Number of connected peers: %d\n", len(p.peerList)-1) // Account for this host
			fmt.Printf("Connected to:\n")
			for id, addrInfo := range p.peerList {
				if id != p.thisHost.ID() {
					fmt.Printf(" - %s (Addresses: %s)\n", id.String(), addrInfo.String())
				}
			}
		} else {
			fmt.Println("No connected peers.")
		}
		p.peerListMutex.Unlock()
		time.Sleep(10 * time.Second) // Announce every 10 seconds
	}
}

// Handle incoming streams
// If it's a new host in the network, assume it's waiting for the peer list. Else, Assume it's a command
func (p *GokerPeer) handleStream(stream network.Stream) {
	defer stream.Close()
	fmt.Println("New stream detected... getting command.")

	// Continuously read and process commands
	reader := bufio.NewReader(stream)

	// Read incoming command
	message, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Error reading from stream: %v", err)
		return
	}
	message = strings.TrimSpace(message)

	// Split the command and the payload
	parts := strings.SplitN(message, " ", 2)
	command := parts[0]
	var payload string
	if len(parts) > 1 {
		payload = parts[1]
	}

	// Process the command based on the message
	switch command {
	case "CMDgetpeers": // Send peerlist to just this stream
		fmt.Println("Recieved peer list request")
		peerList := p.getPeerList()
		_, err := stream.Write([]byte(peerList)) // Ensure the response is newline-terminated
		if err != nil {
			log.Printf("Failed to send peer list: %v", err)
		}
	case "CMDpqrequest":
		fmt.Println("Recieved PQ Request")
		p.RespondToCommand(&PQRequestCommand{}, stream)
	case "CMDstartprotocol":
		fmt.Println("Recieved Start Protocol command")
		p.gameInfo.SetDeck(payload)
		p.RespondToCommand(&StartProtocolCommand{}, stream)
	default:
		log.Printf("Unknown Response Recieved: %s\n", message)
	}
}

// Function to connect to a hosting peer (bootstrapping)
func (p *GokerPeer) connectToHost(peerAddr string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // I believe these seconds indicate how long to wait before stop trying
	defer cancel()

	// Convert the address string to a Multiaddr
	addr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		log.Fatalf("Invalid address: %v", err)
	}

	// Get peer information from the address
	pinfo, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		log.Fatalf("Failed to get peer info: %v", err)
	}

	// Connect to the host
	if err := p.thisHost.Connect(ctx, *pinfo); err != nil {
		log.Fatalf("Failed to connect to bootstrap peer: %v", err)
	}

	fmt.Printf("Connected to host: %s\n", pinfo.ID)
	p.addPeer(pinfo.ID, pinfo.Addrs[0]) // Add host peer to peer list
	p.sessionHost = pinfo.ID            // Set this nodes session host to the bootstrapping host it connected to

	// Create stream with host to call the initial 'CMDgetpeers' and 'CMDpqrequest' commands
	stream, err := p.thisHost.NewStream(ctx, pinfo.ID, protocolID)
	if err != nil {
		log.Fatalf("Failed to create stream: %v", err)
	}
	defer stream.Close()

	// Request the peer list
	_, err = stream.Write([]byte("CMDgetpeers\n"))
	if err != nil {
		log.Fatalf("Failed to send CMDgetpeers command: %v", err)
	}

	// Read the peer list sent by the host
	peerListBytes, err := io.ReadAll(stream)
	if err != nil {
		log.Fatalf("Failed to read peer list: %v", err)
	}

	// Set the peer list and connect to all peers
	p.setPeerList(string(peerListBytes))
	fmt.Println("Received and set peerlist.")

	// Request P & Q upon successful connection and integration with the network
	p.ExecuteCommand(&PQRequestCommand{})
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
}

// Handle existing peer disconnection - Called when the DisconnectF NOTIFICATION has been made
func (p *GokerPeer) handlePeerDisconnection(peerID peer.ID) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	// Remove the peer from the peer list
	if _, exists := p.peerList[peerID]; exists {
		delete(p.peerList, peerID)
		fmt.Printf("Peer %s disconnected and removed from peer list.\n", peerID)
	} else {
		log.Printf("Peer %s not found in peer list.\n", peerID)
	}
}
