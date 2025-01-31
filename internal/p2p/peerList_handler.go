package p2p

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/multiformats/go-multiaddr"
)

// Get a formatted list of peers - ignores self
func (p *GokerPeer) getPeerList() string {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	var peerList string
	for peerID, addr := range p.peerList {
		if peerID != p.thisHost.ID() {
			peerList += fmt.Sprintf("%s %s\n", peerID.String(), addr.String()) // Sends peer list as a multi-line string
		}
	}
	return peerList
}

// Add a peer to the server's list
func (p *GokerPeer) addPeer(peerID peer.ID, addr multiaddr.Multiaddr) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()
	p.peerList[peerID] = addr
}

// Set the peer list (Given the output from the getPeerList function) and connect to all peers
func (p *GokerPeer) setPeerListAndConnect(peerList string) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

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
				if _, exists := p.peerList[pid]; !exists {
					p.peerList[pid] = addr
					// Connect to each peer
					addrInfo := peer.AddrInfo{ID: pid, Addrs: []multiaddr.Multiaddr{addr}}
					if err := p.thisHost.Connect(context.Background(), addrInfo); err != nil {
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
