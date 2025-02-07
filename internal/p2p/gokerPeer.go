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
	ThisHost          host.Host // This host
	ThisHostLBAddress string    // This hosts loopback address (127.0.0.1)
	ThisHostLNAddress string    // This hosts LAN address

	sessionHost   peerInfo   // Host of the current network (This will change has hosts drop out, but will be used to request specific things)
	peerList      []peerInfo // A list of peers in this network - Also used as candidate list and turn order (host will be added on game start)
	peerListMutex sync.Mutex // Mutex for accessing peer map

	// Other
	deck    *deckInfo    // Holds all deck logic (cards, deck operations etc.)
	keyring *sra.Keyring // Holds all encryption logic

	// state given by the game manager
	gameState *gamestate.GameState
}

// Holds important information about other peers in the network
type peerInfo struct {
	ID   peer.ID
	Addr multiaddr.Multiaddr
}

func (p *GokerPeer) Init(nickname string, hosting bool, givenAddr string, givenState *gamestate.GameState) {
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
	p.ThisHost = h

	// Set the givenState
	p.gameState = givenState

	// Add host to state
	p.gameState.AddPeerToState(p.ThisHost.ID(), nickname)

	// Set stream handler for this peer
	h.SetStreamHandler(protocolID, p.handleStream)

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
		// Set host at start of peerlist
		p.peerList = append(p.peerList, peerInfo{ID: p.ThisHost.ID(), Addr: h.Addrs()[4]})
	} else if givenAddr != "" { // Connect to an existing bootstrap server
		fmt.Println("Joining host...")
		p.connectToHost(givenAddr)
	}

	// Start as host listener
	go p.handleNotifications()

	// Handle State changes forever
	go p.handleStateChanges()

	// To tell the game mananger the network is ready to go
	channelmanager.FNET_NetActionDoneChan <- struct{}{}
}

// Handle state changes coming from GUI
func (p *GokerPeer) handleStateChanges() {
	for {
		select {
		case givenAction := <-channelmanager.TNET_ActionChan:
			switch givenAction.Action {
			case "startround": // Populate the state given from the game manager with player info

				// Set starting cash
				for id := range p.gameState.Players {
					p.gameState.PlayersMoney[id] = p.gameState.StartingCash
				}

				// Set Turn Order
				var IDs []peer.ID
				p.peerListMutex.Lock()
				for _, info := range p.peerList {
					IDs = append(IDs, info.ID)
				}
				p.peerListMutex.Unlock()
				p.gameState.SetTurnOrder(IDs)

				p.gameState.WhosTurn = 0 // Start with the dealer for debug

				channelmanager.FNET_NetActionDoneChan <- struct{}{} // Done updating state
			}
		}
	}
}
