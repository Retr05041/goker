package p2p

import (
	"fmt"
	"goker/internal/game"
	"goker/internal/sra"
	"log"
	"sync"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

type GokerPeer struct {
	// Network info
	thisHost          host.Host // This host
	ThisHostMultiaddr string    // This hosts multiaddress

	sessionHost peer.ID // Host of the current network (This will change has hosts drop out, but will be used to request specific things)

	peerList      map[peer.ID]multiaddr.Multiaddr // A map for managing peer connections
	peerListMutex sync.Mutex                      // Mutex for accessing peer map

	// Other
	gameInfo *game.GameInfo // Holds all game info (cards, deck operations etc.)
	keyring  *sra.Keyring   // Pointer to global keyring being used in the game
}

func (p *GokerPeer) Init(hosting bool, givenAddr string) {
	p.keyring = new(sra.Keyring)
	p.gameInfo = new(game.GameInfo)

	// Create a new libp2p Host
	h, err := libp2p.New()
	if err != nil {
		log.Fatalf("failed to create host: %v", err)
	}

	p.thisHost = h
	p.peerList = make(map[peer.ID]multiaddr.Multiaddr)

	// Listen for incoming connections - Use an anonymous function atm since we don't want to do much
	h.SetStreamHandler(protocolID, p.handleStream)

	p.peerList[h.ID()] = h.Addrs()[0]

	// Print the host's ID and multiaddresses
	p.ThisHostMultiaddr = h.Addrs()[0].String() + "/p2p/" + h.ID().String()
	fmt.Printf("Host created. We are: %s\n", h.ID())
	// Green console colour: 	\x1b[32m
	// Reset console colour: 	\x1b[0m
	if hosting {
		fmt.Printf("Listening on specifc multiaddress (give this to other peers): \x1b[32m %s \x1b[0m\n", p.ThisHostMultiaddr)
	}

	if hosting {
		// Start as a bootstrap server
		fmt.Println("Running as a host...")
		p.keyring.GeneratePQ()
		p.keyring.GenerateKeys()
		fmt.Println("Initial keyring ready! awaiting peers.")
		//go s.alert()
	} else if givenAddr != "" {
		// Connect to an existing bootstrap server
		fmt.Println("Joining host...")
		p.connectToHost(givenAddr)
		//go s.alert()
	}

	// Handle notifications forever
	go p.handleNotifications()
}
