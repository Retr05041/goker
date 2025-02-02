package p2p

import (
	"context"
	"goker/internal/channelmanager"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// 'peer' connection handler

// Function to connect to a hosting peer (bootstrapping)
// On success peer should be added to the network and know about all current peers
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
	if err := p.ThisHost.Connect(ctx, *pinfo); err != nil {
		log.Fatalf("Failed to connect to bootstrap peer: %v", err)
	}

	log.Printf("Connected to host: %s\n", pinfo.ID)

	// Set sessionHost
	p.sessionHost = peerInfo{ID: pinfo.ID, Addr: addr}

	// Get and set peerlist from host
	p.ExecuteCommand(&GetPeerListCommand{})

	// Tell GUI to change the number of players
	channelmanager.FNET_NumOfPlayersChan <- len(p.peerList)

	// Request Nicknames and add them to the state
	p.ExecuteCommand(&NicknameRequestCommand{})
}
