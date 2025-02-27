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

// Returns a list of `peerID peerAddr\n` for every peer in the peer list in order
func (p *GokerPeer) getPeerList() string {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	var peerList []string
	for _, peerInfo := range p.peerList {
		peerList = append(peerList, fmt.Sprintf("%s %s", peerInfo.ID.String(), peerInfo.Addr.String()))
	}
	return strings.Join(peerList, "\n")
}

// Set the peer list (Given the output from the getPeerList function) and connect to all new peers
func (p *GokerPeer) setPeerListAndConnect(peerList string) {

	var sentPeerList []peerInfo
	scanner := bufio.NewScanner(strings.NewReader(peerList))

	for scanner.Scan() {
		parts := strings.Fields(scanner.Text()) // Handles spaces in the line
		if len(parts) < 2 {
			continue
		}

		newPeerID, err := peer.Decode(parts[0]) // make it a peerID
		if err != nil {
			log.Fatal("setPeerList: Unable to decode incoming peerID")
		}

		addr, err := multiaddr.NewMultiaddr(parts[1]) // Make it a multiaddr
		if err != nil {
			log.Fatal("setPeerList: Unable to create multiaddr for incoming peerID - " + newPeerID.String())
		}

		sentPeerList = append(sentPeerList, peerInfo{ID: newPeerID, Addr: addr})
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("setPeerList: Error reading peer list - %v", err)
	}

	// Time to set it

	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	existing := make(map[peerInfo]struct{}, len(p.peerList))
	for _, peerInfo := range p.peerList {
		existing[peerInfo] = struct{}{}
	}

	// Add new peers and connect to them
	for _, newPeerInfo := range sentPeerList {
		if _, found := existing[newPeerInfo]; !found {
			p.peerList = append(p.peerList, newPeerInfo)
			existing[newPeerInfo] = struct{}{} // Prevent duplicate additions

			// Connect to each peer new peer
			if newPeerInfo.ID != p.ThisHost.ID() {
				addrInfo := peer.AddrInfo{ID: newPeerInfo.ID, Addrs: []multiaddr.Multiaddr{newPeerInfo.Addr}}
				if err := p.ThisHost.Connect(context.Background(), addrInfo); err != nil {
					log.Printf("Failed to connect to peer %s: %v", newPeerInfo.ID.String(), err)
				} else {
					fmt.Printf("Connected to new peer: %s", newPeerInfo.ID.String())
				}
			}
		}
	}
}
