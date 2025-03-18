package p2p

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"goker/internal/channelmanager"
	"goker/internal/gamestate"
	"goker/internal/sra"
	"log"
	"math/big"
	"sync"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

var protocolID = protocol.ID("/goker/command/1.0.0")

type GokerPeer struct {
	// Network logic
	ThisHost          host.Host // This host
	ThisHostLBAddress string    // This hosts loopback address (127.0.0.1)
	ThisHostLNAddress string    // This hosts LAN address

	sessionHost   peerInfo   // Host of the current network (This will change has hosts drop out, but will be used to request specific things)
	peerList      []peerInfo // A list of peers in this network - Also used as candidate list and turn order (host will be added on game start)
	peerListMutex sync.Mutex // Mutex for accessing peer map

	// Other
	Deck        *deckInfo // Holds all deck logic (cards, deck operations etc.)
	OthersHands map[peer.ID][]*CardInfo
	MyHand      []*CardInfo // Holds my hand
	Flop        []*CardInfo // Holds flop
	Turn        *CardInfo
	River       *CardInfo
	Keyring     *sra.Keyring // Holds all encryption logic

	// state given by the game manager
	gameState *gamestate.GameState

	// Context (tag) for commands per betting phase
	tag uint64
}

// Holds important information about other peers in the network
type peerInfo struct {
	ID   peer.ID
	Addr multiaddr.Multiaddr
}

func (p *GokerPeer) Init(nickname string, hosting bool, givenAddr string, givenState *gamestate.GameState) {
	// Setup deck and keyring for later
	p.Keyring = new(sra.Keyring)
	p.Keyring.GenerateSigningKeys()

	p.Deck = new(deckInfo)
	p.OthersHands = make(map[peer.ID][]*CardInfo)
	// TODO: Make this decided at runtime?
	p.Deck.GenerateDecks("gokerdecksecretkeyforhashesversion1")

	// Set the givenState
	p.gameState = givenState

	// Create a new libp2p Host
	h, err := libp2p.New()
	if err != nil {
		log.Fatalf("failed to create host: %v", err)
	}
	// Setup this Host
	p.ThisHost = h
	// Add host to state
	p.gameState.AddPeerToState(p.ThisHost.ID(), nickname)
	p.gameState.Me = p.ThisHost.ID()
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

	// To tell the game mananger the network is ready to go
	channelmanager.FNET_NetActionDoneChan <- struct{}{}
}

func (p *GokerPeer) SetTurnOrderWithLobby() {
	// Set Turn Order
	var IDs []peer.ID
	p.peerListMutex.Lock()
	for _, info := range p.peerList {
		IDs = append(IDs, info.ID)
	}
	p.peerListMutex.Unlock()
	p.gameState.SetTurnOrder(IDs)
}

// Unlocks the time-locked key by performing
// `t` sequential squaring operations.
func (p *GokerPeer) BreakTimeLockedPuzzle(peerID peer.ID, puzzlePayload []byte) {
	var message sra.TimeLock
	if err := json.Unmarshal([]byte(puzzlePayload), &message); err != nil {
		fmt.Println("Failed to parse received time-lock puzzle:", err)
		return
	}

	// Convert string fields to big.Int
	puzzle, _ := new(big.Int).SetString(message.Puzzle, 10)
	iterations, _ := new(big.Int).SetString(message.Iter, 10)
	n, _ := new(big.Int).SetString(message.N, 10)

	fmt.Printf("Received time-locked puzzle from %s, beginning decryption...\n", &message)

	// Step 1: Set base
	base := big.NewInt(2)

	// Step 2: Perform 't' squarings of 'base' modulo 'n'
	for i := big.NewInt(0); i.Cmp(iterations) < 0; i.Add(i, big.NewInt(1)) {
		base.Exp(base, big.NewInt(2), n)
	}

	// Step 3: Subtract `b` from the time locked puzzle to retrieve the private key
	key := new(big.Int).Sub(puzzle, base)
	key.Mod(key, n)

	plaintextPayload, err := sra.AESToPayload(message.Payload, key)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("PUZZLE BROKE FOR: " + peerID.String())
	p.Keyring.BrokenPuzzlePayloads = append(p.Keyring.BrokenPuzzlePayloads, plaintextPayload)

	// Signal that a puzzle was broken
	p.gameState.NumOfPuzzlesBroken++
	channelmanager.TGM_WaitForPuzzles <- struct{}{}
}

func (p *GokerPeer) GenerateNewTag() {
	err := binary.Read(rand.Reader, binary.LittleEndian, &p.tag)
	if err != nil {
		log.Fatalf("GenerateNewTag: failed to generate random tag: %s\n", err.Error())
	}
}

func (p *GokerPeer) SetNewTag(tag uint64) {
	p.tag = tag
}
