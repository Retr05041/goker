package p2p

import (
	"strings"
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"bufio"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/multiformats/go-multiaddr"
)

var protocolID = protocol.ID("/goker/1.0.0")


type BootstrapServer struct {
    host   host.Host
    peers  map[peer.ID]multiaddr.Multiaddr
    mutex  sync.Mutex
}

var server BootstrapServer

func Init(hosting bool, givenAddr string) {
    // Create a new libp2p Host
    h, err := libp2p.New()
    if err != nil {
        log.Fatalf("failed to create host: %v", err)
    }

    server = BootstrapServer{
        host:  h,
        peers: make(map[peer.ID]multiaddr.Multiaddr),
    }

    // Listen for incoming connections - Use an anonymous function atm since we don't want to do much
    h.SetStreamHandler(protocolID, server.handleStream) 

	server.peers[h.ID()] = h.Addrs()[0]

    // Print the host's ID and multiaddresses
    fmt.Printf("Host created. We are: %s\n", h.ID())
	// Green console colour: 	\x1b[32m
	// Reset console colour: 	\x1b[0m
	fmt.Printf("Listening on specifc multiaddress (give this to other peers): \x1b[32m %s/p2p/%s \x1b[0m\n", h.Addrs()[0].String(), h.ID())

    if hosting {
        // Start as a bootstrap server
        fmt.Println("Running as bootstrap server...")
		go alert()
    } else if givenAddr != "" {
        // Connect to an existing bootstrap server
        fmt.Println("Joining bootstrap server...")
        server.connectToBootstrapPeer(givenAddr)
		go alert()
    }

    // Handle new peers being discovered (connected to this host, and disconnected from this host)
    h.Network().Notify(&network.NotifyBundle{
        ConnectedF: func(n network.Network, conn network.Conn) { // On peer connect
			log.Printf("NOTIFICATION: Connection from new peer: %s\n", conn.RemotePeer()) // WHEN A NEW PEER CONNECTS TO US, IT MUST BE FROM A BROADCAST SERVER SENDING IT

			// Connect to the new peer and update the peer list
			server.connectToNewPeer(conn.RemotePeer(), conn.RemoteMultiaddr())
        },
        DisconnectedF: func(n network.Network, conn network.Conn) { // On peer disconnect
			log.Printf("NOTIFICATION: Disconnected from peer: %s\n", conn.RemotePeer())

			server.handlePeerDisconnection(conn.RemotePeer())
        },
    })

    // Keep the program running
    select {}
}

// Log connected peers - Ignore self
func alert() {
	log.Printf("LOGGING HAS BEGUN")
    for {
        server.mutex.Lock()
        if len(server.peers) > 1 { // Account for this host
			log.Printf("Number of connected peers: %d\n", len(server.peers)-1) // Account for this host
            fmt.Printf("Connected to:\n")
            for id, addrInfo := range server.peers {
				if id != server.host.ID() {
					fmt.Printf(" - %s (Addresses: %s)\n", id.String(), addrInfo.String())
				}
            }
        } else {
            fmt.Println("No connected peers.")
        }
        server.mutex.Unlock()
        time.Sleep(10 * time.Second) // Announce every 10 seconds
    }
}

// Handle incoming peer connections 
// Aquires peer info - updates peerlist - Sends peer new peer list - broadcasts change to all known peers (Excluding newly added peer and self) 
func (s *BootstrapServer) handleStream(stream network.Stream) {
    defer stream.Close()
	log.Println("New stream detected... handling now")

    // Get the new peer's ID
    newPeerID := stream.Conn().RemotePeer()
    s.mutex.Lock()
    s.peers[newPeerID] = stream.Conn().RemoteMultiaddr()
    s.mutex.Unlock()
	log.Println("handleStream: Peerlist updated")

    // Send the peer list to the newly connected peer
	peerList := s.getPeerList()
    _, err := stream.Write([]byte(peerList))
    if err != nil {
		log.Printf("handleStream: Failed to send peer list: %v", err)
        return
    }
	log.Println("Peer list sent to new peer - Bootstrapping complete!")

    // Broadcast the new peer to all connected peers
	log.Println("Broadcasting change to existing peers... - IMPLEMENT")
}

// Get a formatted list of peers - ignores self
func (s *BootstrapServer) getPeerList() string {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    var peerList string
    for peerID, addr := range s.peers {
		if peerID != server.host.ID() {
			peerList += fmt.Sprintf("%s %s\n",peerID.String(), addr.String()) // Sends peer list as a multi-line string
		}
    }
    return peerList
}

// Function to connect to a bootstrap peer - Only used when joining a network through a bootstrapping peer
func (s *BootstrapServer) connectToBootstrapPeer(peerAddr string) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

    // Connect to the bootstrap peer
    if err := s.host.Connect(ctx, *pinfo); err != nil {
        log.Fatalf("Failed to connect to bootstrap peer: %v", err)
    }

    fmt.Printf("Connected to bootstrap peer: %s\n", pinfo.ID)
	server.mutex.Lock()
	server.peers[pinfo.ID] = pinfo.Addrs[0]
	server.mutex.Unlock()

    // Wait for peer list from the bootstrap server
    stream, err := s.host.NewStream(ctx, pinfo.ID, protocolID)
    if err != nil {
        log.Fatalf("Failed to create stream: %v", err)
    }
    defer stream.Close()

    // Read the peer list sent by the bootstrap server
    reader := bufio.NewReader(stream)
    peerList, err := reader.ReadString('\n')
    if err != nil {
        log.Fatalf("Failed to read peer list: %v", err)
    }

    fmt.Printf("Received peer list:\n%s", peerList)

    // Set the peer list and connect to all peers
    s.setPeerList(peerList)
}

// Add a peer to the server's list
func (s *BootstrapServer) addPeer(peerID peer.ID, addr multiaddr.Multiaddr) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.peers[peerID] = addr
}


// Set the peer list (Given the output from the getPeerList function) and connect to all peers
func (s *BootstrapServer) setPeerList(peerList string) {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    scanner := bufio.NewScanner(strings.NewReader(peerList)) // Put it in a scanner for processing
    for scanner.Scan() { // Next token
        line := scanner.Text() // Get the current line seperated by \n
		parts := strings.Split(line, " ") // Split them to peer ID - Multiaddr
		if len(parts) < 2 {
			continue
		}
        pid, err := peer.Decode(parts[0]) // make it a peerID
        if err == nil {
			addr, err := multiaddr.NewMultiaddr(parts[1]) // Make it a multiaddr
			if err == nil {
				if _, exists := s.peers[pid]; !exists {
					s.peers[pid] = addr
					// Connect to each peer
					addrInfo := peer.AddrInfo{ID: pid, Addrs: []multiaddr.Multiaddr{addr}}
					if err := s.host.Connect(context.Background(), addrInfo); err != nil {
						log.Printf("Failed to connect to peer %s: %v", pid, err)
					} else {
						log.Printf("Connected to new peer: %s", pid)
					}
				}
			} else {
				log.Printf("setPeerList: Unable to create multiaddr for incoming peerID - " + pid.String())
			}
        } else {
			log.Printf("setPeerList: Unable to decode incoming peerID")
		}
    }

    if err := scanner.Err(); err != nil {
		log.Printf("setPeerList: Error reading peer list - %v", err)
    }
}

// Connect to new peers that are discovered - Called when the ConnectF NOTIFICATION has been made
func (s *BootstrapServer) connectToNewPeer(newPeerID peer.ID, newPeerAddr multiaddr.Multiaddr) {
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
    log.Printf("Connected to new peer: %s\n", newPeerID)
}

// Handle peer disconnection - Called when the DisconnectF NOTIFICATION has been made
func (s *BootstrapServer) handlePeerDisconnection(peerID peer.ID) {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    // Remove the peer from the peer list
    if _, exists := s.peers[peerID]; exists {
        delete(s.peers, peerID)
        log.Printf("Peer %s disconnected and removed from peer list.\n", peerID)
    } else {
        log.Printf("Peer %s not found in peer list.\n", peerID)
    }
}
