package p2p

import (
	"fmt"
	"goker/internal/channelmanager"
	"goker/internal/gamestate"
	"goker/internal/sra"
	"log"
	"sync"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

var protocolID = protocol.ID("/goker/1.0.0")

type GokerPeer struct {
	// Network logic
	thisHost          host.Host // This host
	ThisHostLBAddress string    // This hosts loopback address (127.0.0.1)
	ThisHostLNAddress string    // This hosts LAN address

	sessionHost   peer.ID   // Host of the current network (This will change has hosts drop out, but will be used to request specific things)
	candidateList []peer.ID // Add peer ID's as they connect :: Used for who's hosting next and turn order

	peerList      map[peer.ID]multiaddr.Multiaddr // A map for managing peer connections
	peerListMutex sync.Mutex                      // Mutex for accessing peer map

	// Other
	deck    *deckInfo    // Holds all deck logic (cards, deck operations etc.)
	keyring *sra.Keyring // Holds all encryption logic

	// Network copy of gamestate
	gameState *gamestate.GameState
}

func (p *GokerPeer) Init(nickname string, hosting bool, givenAddr string) {
	// TODO: Move these to later on when using the keyring is necessary
	p.keyring = new(sra.Keyring)
	p.deck = new(deckInfo)
	p.deck.GenerateRefDeck("mysupersecretkey")
	p.deck.GenerateRoundDeck("mysupersecretkey")

	// Create a new libp2p Host
	h, err := libp2p.New()
	if err != nil {
		log.Fatalf("failed to create host: %v", err)
	}

	// Setup this Host
	p.thisHost = h
	p.peerList = make(map[peer.ID]multiaddr.Multiaddr)

	// Setup gamestate
	p.gameState = new(gamestate.GameState)
	p.gameState.Players = make(map[peer.ID]string)
	p.gameState.Players[h.ID()] = nickname
	p.gameState.PlayersMoney = make(map[string]float64)
	p.gameState.BetHistory = make(map[string]float64)

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

	if hosting { // Start as a bootstrap server
		fmt.Println("Running as a host...")
		p.keyring.GeneratePQ() // Generate first shared p and q
		p.keyring.GenerateKeys()
		fmt.Println("Initial keyring ready! awaiting peers.")
	} else if givenAddr != "" { // Connect to an existing bootstrap server
		fmt.Println("Joining host...")
		p.connectToHost(givenAddr)
	}

	// Handle State changes forever
	go p.handleStateChanges()

	// Start as host
	go p.handleNotifications()

	// To tell the game mananger the network is ready to go
	channelmanager.FNET_NetActionDoneChan <- struct{}{}
}

// TODO: Implement handleStateChanges
// Handles state changes on the local side
func (p *GokerPeer) handleStateChanges() {
	for {
		select {
		case givenAction := <-channelmanager.TFNET_GameStateChan:
			switch givenAction.Action {
			case "startround": // Set: Nickanes -> Peer ID's | Turn Order ->
				p.gameState.MinBet = givenAction.State.MinBet
				p.gameState.Pot = givenAction.State.Pot
				p.gameState.Phase = givenAction.State.Phase
				p.gameState.StartingCash = givenAction.State.StartingCash

				// Set starting cash
				for _, nick := range p.gameState.Players {
					p.gameState.PlayersMoney[nick] = p.gameState.StartingCash
				}

				// For debug when the round starts
				p.gameState.DumpState()
				// TODO: Send gamestate to everyone
				// TODO: Run command for everyone to move to the game view
			}
		}
	}
}
