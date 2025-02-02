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

// Returns a list of `peerID, peerAddr\n` for every peer in the peer list in order
func (p *GokerPeer) getPeerList() string {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	var peerList string
	for _, peerInfo := range p.peerList {
		peerList += fmt.Sprintf("%s %s\n", peerInfo.ID.String(), peerInfo.Addr.String()) // Sends peer list as a multi-line string
	}
	return peerList
}

// Set the peer list (Given the output from the getPeerList function) and connect to all new peers
func (p *GokerPeer) setPeerListAndConnect(peerList string) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	var sentPeerList []peerInfo

	scanner := bufio.NewScanner(strings.NewReader(peerList)) // Put it in a scanner for processing
	for scanner.Scan() {                                     // Next token
		line := scanner.Text()            // Get the current line seperated by \n
		parts := strings.Split(line, " ") // Split them to peer ID - Multiaddr
		if len(parts) < 2 {
			continue
		}

		newPeerID, err := peer.Decode(parts[0]) // make it a peerID
		if err != nil {
			log.Printf("setPeerList: Unable to decode incoming peerID")
		}

		addr, err := multiaddr.NewMultiaddr(parts[1]) // Make it a multiaddr
		if err != nil {
			log.Println("setPeerList: Unable to create multiaddr for incoming peerID - " + newPeerID.String())
		}

		sentPeerList = append(sentPeerList, peerInfo{ID: newPeerID, Addr: addr})
	}

		
	existing := make(map[peerInfo]struct{}, len(p.peerList))
	for _, peerInfo := range p.peerList {
		existing[peerInfo] = struct{}{}
	}

	// Add new peers and connect to them
	for _, newPeerInfo := range sentPeerList {
		if _, found := existing[newPeerInfo]; !found {
			p.peerList = append(p.peerList, newPeerInfo)
			
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


	if err := scanner.Err(); err != nil {
		log.Printf("setPeerList: Error reading peer list - %v", err)
	}
}
