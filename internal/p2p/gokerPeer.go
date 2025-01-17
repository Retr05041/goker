package p2p

import (
	"fmt"
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
	ThisHostMultiaddr string                          // This hosts multiaddress
	sessionHost       peer.ID                         // Host of the current network (This will change has hosts drop out, but will be used to request specific things)
	peerList          map[peer.ID]multiaddr.Multiaddr // A map for managing peer connections
	peerListMutex     sync.Mutex                      // Mutex for accessing peer map

	// Other
	deck    *deckInfo    // Holds all deck logic (cards, deck operations etc.)
	keyring *sra.Keyring // Holds all encryption logic
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

func (g *GokerPeer) DisplayDeck() {
	g.deck.DisplayDeck()
}

// Decrypt deck with global keys
func (p *GokerPeer) DecryptAllWithGlobalKeys() {
	for _, card := range p.deck.RoundDeck {
		card.Cardvalue = p.keyring.DecryptWithGlobalKeys(card.Cardvalue)
	}
}

// Encrypt deck with global keys
func (p *GokerPeer) EncryptAllWithGlobalKeys() {
	for _, card := range p.deck.RoundDeck {
		card.Cardvalue = p.keyring.EncryptWithGlobalKeys(card.Cardvalue)
	}
}

// Encrypt deck with variation numbers
func (p *GokerPeer) EncryptAllWithVariation() {
	for i, card := range p.deck.RoundDeck {
		encryptedCard, err := p.keyring.EncryptWithVariation(card.Cardvalue, i)
		if err != nil {
			log.Println(err)
		}
		p.deck.RoundDeck[i].Cardvalue = encryptedCard
		p.deck.RoundDeck[i].VariationIndex = i
	}
}
