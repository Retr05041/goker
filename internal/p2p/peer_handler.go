package p2p

import (
	"fmt"
	"bufio"
	"context"
	"log"
	"strings"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/multiformats/go-multiaddr"
)

// Get a formatted list of peers - ignores self
func (s *BootstrapServer) getPeerList() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var peerList string
	for peerID, addr := range s.peers {
		if peerID != s.host.ID() {
			peerList += fmt.Sprintf("%s %s\n", peerID.String(), addr.String()) // Sends peer list as a multi-line string
		}
	}
	return peerList
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
	for scanner.Scan() {                                     // Next token
		line := scanner.Text()            // Get the current line seperated by \n
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
						fmt.Printf("Connected to new peer: %s", pid)
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
