package p2p

import (
	"fmt"
	"goker/internal/channelmanager"
	"goker/internal/sra"
	"log"
	"sync"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

type GokerPeer struct {
	// Network logic
	thisHost          host.Host                       // This host
	ThisHostLBAddress string                          // This hosts loopback address (127.0.0.1)
	ThisHostLNAddress string                          // This hosts LAN address
	sessionHost       peer.ID                         // Host of the current network (This will change has hosts drop out, but will be used to request specific things)
	peerList          map[peer.ID]multiaddr.Multiaddr // A map for managing peer connections
	peerListMutex     sync.Mutex                      // Mutex for accessing peer map
	// Other
	deck    *deckInfo    // Holds all deck logic (cards, deck operations etc.)
	keyring *sra.Keyring // Holds all encryption logic

	// Data accessable to the gamemanager
	Nickname  string
	Nicknames []string // Everyone else's nicknames, in candidate list order to match with
}

func (p *GokerPeer) Init(hosting bool, givenAddr string) {
	p.keyring = new(sra.Keyring)
	p.deck = new(deckInfo)
	p.deck.GenerateRefDeck("mysupersecretkey")
	p.deck.GenerateRoundDeck("mysupersecretkey")

	// Create a new libp2p Host
	h, err := libp2p.New()
	if err != nil {
		log.Fatalf("failed to create host: %v", err)
	}

	p.thisHost = h
	p.peerList = make(map[peer.ID]multiaddr.Multiaddr)

	// Listen for incoming connections - Use an anonymous function atm since we don't want to do much
	h.SetStreamHandler(protocolID, p.handleStream)

	p.peerList[h.ID()] = h.Addrs()[4] // base peerList off of lan IP's till we go international

	// Print the host's ID and multiaddresses
	p.ThisHostLBAddress = h.Addrs()[0].String() + "/p2p/" + h.ID().String()
	p.ThisHostLNAddress = h.Addrs()[4].String() + "/p2p/" + h.ID().String()
	fmt.Printf("Host created. We are: %s\n", h.ID())
	// Green console colour: 	\x1b[32m
	// Reset console colour: 	\x1b[0m
	if hosting {
		fmt.Printf("Listening on specifc multiaddress (give this to other peers): \x1b[32m %s \x1b[0m\n", p.ThisHostLNAddress)
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

	channelmanager.NetworkInitDoneChannel <- struct{}{}
	// Handle notifications forever
	go p.handleNotifications()
}

func (g *GokerPeer) DisplayDeck() {
	g.deck.DisplayDeck()
}
