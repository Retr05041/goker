package p2p

import (
	"bufio"
	"context"
	"fmt"
	"goker/internal/sra"
	"io"
	"log"
	"strings"
	"sync"
	"time"
	"math/big"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/multiformats/go-multiaddr"
)

var protocolID = protocol.ID("/goker/1.0.0")

type BootstrapServer struct {
	host          host.Host // This host
	HostMultiaddr string // This hosts multiaddress

	peers         map[peer.ID]multiaddr.Multiaddr // A map for managing peer connections
	mutex         sync.Mutex // Mutex for accessing peer map

	keyring sra.Keyring // Keyring for handling encryption
	sessionHost peer.ID // Host of the current network (This will change has hosts drop out, but will be used to request specific things)
}

func (s *BootstrapServer) Init(hosting bool, givenAddr string) {
	// Create a new libp2p Host
	h, err := libp2p.New()
	if err != nil {
		log.Fatalf("failed to create host: %v", err)
	}

	s.host = h
	s.peers = make(map[peer.ID]multiaddr.Multiaddr)

	// Listen for incoming connections - Use an anonymous function atm since we don't want to do much
	h.SetStreamHandler(protocolID, s.handleStream)

	s.peers[h.ID()] = h.Addrs()[0]

	// Print the host's ID and multiaddresses
	s.HostMultiaddr = h.Addrs()[0].String() + "/p2p/" + h.ID().String()
	fmt.Printf("Host created. We are: %s\n", h.ID())
	// Green console colour: 	\x1b[32m
	// Reset console colour: 	\x1b[0m
	if hosting {
		fmt.Printf("Listening on specifc multiaddress (give this to other peers): \x1b[32m %s \x1b[0m\n", s.HostMultiaddr)
	}

	if hosting {
		// Start as a bootstrap server
		fmt.Println("Running as a host... setting up initial keyring")
		s.keyring.GeneratePQ()
		s.keyring.GenerateGlobalKeys()
		fmt.Println("Done, reading for peers!")
		//go s.alert()
	} else if givenAddr != "" {
		// Connect to an existing bootstrap server
		fmt.Println("Joining host...")
		s.connectToHost(givenAddr)
		//go s.alert()
	}

	// Handle notifications forever
	go s.handleNotifications()
}

func (s *BootstrapServer) handleNotifications() {
	// Handle new peers being discovered (connected to this host, and disconnected from this host)
	s.host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, conn network.Conn) { // On peer connect
			fmt.Printf("NOTIFICATION: Connection from new peer: %s\n", conn.RemotePeer()) // WHEN A NEW PEER CONNECTS TO US, IT MUST BE FROM A BROADCAST SERVER SENDING IT

			// Connect to the new peer and update the peer list
			s.handlePeerConnection(conn.RemotePeer(), conn.RemoteMultiaddr())
		},
		DisconnectedF: func(n network.Network, conn network.Conn) { // On peer disconnect
			fmt.Printf("NOTIFICATION: Disconnected from peer: %s\n", conn.RemotePeer())

			s.handlePeerDisconnection(conn.RemotePeer())
		},
	})

	// Run this function forever - IF YOU REMOVE THIS, THE PROGRAM WILL CLOSE
	select {}
}

// Log connected peers - Ignore self - For debugging only
func (s *BootstrapServer) alert() {
	fmt.Println("LOGGING HAS BEGUN")
	for {
		s.mutex.Lock()
		if len(s.peers) > 1 { // Account for this host
			fmt.Printf("Number of connected peers: %d\n", len(s.peers)-1) // Account for this host
			fmt.Printf("Connected to:\n")
			for id, addrInfo := range s.peers {
				if id != s.host.ID() {
					fmt.Printf(" - %s (Addresses: %s)\n", id.String(), addrInfo.String())
				}
			}
		} else {
			fmt.Println("No connected peers.")
		}
		s.mutex.Unlock()
		time.Sleep(10 * time.Second) // Announce every 10 seconds
	}
}

// Handle incoming streams
// If it's a new host in the network, assume it's waiting for the peer list. Else, Assume it's a command
func (s *BootstrapServer) handleStream(stream network.Stream) {
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
	case "CMDping":
		fmt.Println("Received ping command")
		s.RespondToCommand(&PingCommand{}, stream)
	case "CMDgetpeers": // Send peerlist to just this stream
		fmt.Println("Recieved peer list request")
		peerList := s.getPeerList()
		_, err := stream.Write([]byte(peerList)) // Ensure the response is newline-terminated
		if err != nil {
			log.Printf("Failed to send peer list: %v", err)
		}
	case "CMDpqrequest":
		fmt.Println("Recieved PQ Request")
		s.RespondToCommand(&PQRequestCommand{}, stream)
	case "CMDencrypt":
		fmt.Println("Recieved Encryption Request on a payload")
		messageBig := new(big.Int)
		messageBig.SetString(payload, 10)
		encryptedMessage := s.keyring.EncryptWithGlobalKeys(messageBig)
		stream.Write([]byte(encryptedMessage.String() + "\n"))
	case "CMDdecrypt":
		fmt.Println("Recieved Decryption Request on a payload")
		messageBig := new(big.Int)
		messageBig.SetString(payload, 10)
		decryptedMessage := s.keyring.DecryptWithGlobalKeys(messageBig)
		stream.Write([]byte(decryptedMessage.String() + "\n"))
	default:
		log.Printf("Unknown Response Recieved: %s\n", message)
	}
}


// Function to connect to a hosting peer (bootstrapping)
func (s *BootstrapServer) connectToHost(peerAddr string) {
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
	if err := s.host.Connect(ctx, *pinfo); err != nil {
		log.Fatalf("Failed to connect to bootstrap peer: %v", err)
	}

	fmt.Printf("Connected to host: %s\n", pinfo.ID)
	s.addPeer(pinfo.ID, pinfo.Addrs[0]) // Add host peer to peer list
	s.sessionHost = pinfo.ID // Set this nodes session host to the bootstrapping host it connected to

	// Create stream with host to call the initial 'CMDgetpeers' and 'CMDpqrequest' commands
	stream, err := s.host.NewStream(ctx, pinfo.ID, protocolID)
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
	s.setPeerList(string(peerListBytes))
	fmt.Println("Received and set peerlist.")

	// Request P & Q upon successful connection and integration with the network
	s.ExecuteCommand(&PQRequestCommand{})	
}


// Connect to new peers that are discovered - Called when the ConnectF NOTIFICATION has been made.
func (s *BootstrapServer) handlePeerConnection(newPeerID peer.ID, newPeerAddr multiaddr.Multiaddr) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if already connected
	if _, exists := s.peers[newPeerID]; exists {
		log.Printf("Already connected to peer: %s\n", newPeerID)
		return
	}

	// Add new peer to the peer list
	s.peers[newPeerID] = newPeerAddr

	// Create address info for the new peer
	addrInfo := peer.AddrInfo{ID: newPeerID, Addrs: []multiaddr.Multiaddr{newPeerAddr}}

	// Attempt to connect to the new peer
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.host.Connect(ctx, addrInfo); err != nil {
		log.Printf("Failed to connect to new peer %s: %v", newPeerID, err)
		return
	}
	fmt.Printf("Connected to new peer: %s\n", newPeerID)
}

// Handle existing peer disconnection - Called when the DisconnectF NOTIFICATION has been made
func (s *BootstrapServer) handlePeerDisconnection(peerID peer.ID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Remove the peer from the peer list
	if _, exists := s.peers[peerID]; exists {
		delete(s.peers, peerID)
		fmt.Printf("Peer %s disconnected and removed from peer list.\n", peerID)
	} else {
		log.Printf("Peer %s not found in peer list.\n", peerID)
	}
}
