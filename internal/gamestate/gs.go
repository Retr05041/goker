package gamestate

import (
	"fmt"
	"goker/internal/channelmanager"
	"log"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

type GameState struct {
	// Gamestate mutex due to the network and gamemanager using the same state
	mu sync.Mutex
	// Handled by network (built dynamically and 'should be' signed by all peers for validity)
	// On peer connection
	Players      map[peer.ID]string  // Player nicknames tied to their peer.ID
	BetHistory   map[peer.ID]float64 // A map to store bets placed on the current round

	// On play being pressed
	PlayersMoney map[peer.ID]float64 // Players money by peer ID
	TurnOrder map[int]peer.ID // Handled by network (based off of candidate list)
	WhosTurn  int

	// Handled by host
	StartingCash float64 // Starting cash for all players
	Pot          float64 // The current pot amount
	MinBet       float64 // Minimum bet required for the round (again from table settings)
	Phase        string  // Current phase of the game (e.g., "preflop", "flop", "turn", "river")
}


// Refresh state for new possible rounds
func (gs *GameState) FreshState(startingCash *float64, minBet *float64) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.Phase = "preflop"
	gs.Pot = 0.0

	gs.StartingCash = 100.0
	if startingCash != nil {
		gs.StartingCash = *startingCash
	}

	gs.MinBet = 1.0
	if minBet != nil {
		gs.MinBet = *minBet
	}
}

// For adding a new peer to the state
func (gs *GameState) AddPeerToState(peerID peer.ID, nickname string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if _, v := gs.Players[peerID]; v {
		log.Println("AddPeerToState: Peer already in state")
		return
	}
	gs.Players[peerID] = nickname
	gs.BetHistory[peerID] = 0.0
}

func (gs *GameState) RemovePeerFromState(peerID peer.ID) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	_, exists := gs.Players[peerID]
	if !exists {
		log.Println("RemovePeerFromState: Peer not in state")
		return
	}
	delete(gs.Players, peerID)
	// Remove bet history
	delete(gs.BetHistory, peerID)

	_, exists = gs.PlayersMoney[peerID]
	if !exists {
		log.Println("RemovePeerFromState: Peer doesn't have money")
		return
	}
	delete(gs.PlayersMoney, peerID)
}

// Get the current pot from the rounds bet history
func (gs *GameState) GetCurrentPot() float64 {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	pot := 0.0
	for _, b := range gs.BetHistory {
		pot += b
	}
	return pot
}

// Sets turn order - in order of given IDs
func (gs *GameState) SetTurnOrder(IDs []peer.ID) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	for i, v := range IDs {
		gs.TurnOrder[i] = v
	}
}

// Check if a player exists
func (gs *GameState) PlayerExists(id peer.ID) bool {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if _, exists := gs.Players[id]; exists { 
		return true
	}
	return false
}

// Get the nickname of a specific player
func (gs *GameState) GetNickname(id peer.ID) string {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	return gs.Players[id]
}

// Formatted player info to be sent to the GUI
func (gs *GameState) GetPlayerInfo() channelmanager.PlayerInfo {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	var players []string
	var money []float64
	for i := 0; i < len(gs.TurnOrder); i++ { // We disregard any 'exists' stuff as by this point we have already locked in everyone
		peerID, _ := gs.TurnOrder[i]	
		peerNickname, _ := gs.Players[peerID]
		peerMoney, _ := gs.PlayersMoney[peerID]
		players = append(players, peerNickname)
		money = append(money, peerMoney)
	}

	return channelmanager.PlayerInfo{Players: players, Money: money}
}

func (gs *GameState) DumpState() {
	fmt.Println("DUMPING STATE")
	fmt.Println("---------------------------------")
	log.Print("Players: ")
	for id, val := range gs.Players {
		fmt.Println(id.String() + ", " + val)
	}
	fmt.Println("---------")
	log.Print("PlayersMoney: ")
	for id, val := range gs.PlayersMoney {
		fmt.Printf("%s, %.1f\n", id, val)

	}
	fmt.Println("---------")
	log.Print("BetHistory: ")
	for id, val := range gs.BetHistory {
		fmt.Printf("%s, %.1f\n", id, val)

	}
	fmt.Println("---------")
	log.Print("TurnOrder: ")
	for i := 0; i < len(gs.TurnOrder); i++ {
		id, _ := gs.TurnOrder[i]
		fmt.Printf("%d. %s\n", i, id.String())
	}
	fmt.Println("---------")
	log.Printf("WhosTurn: %d", gs.WhosTurn)
	fmt.Println("---------")
	log.Printf("StartingCash: %.1f", gs.StartingCash)
	fmt.Println("---------")
	log.Printf("Pot: %.1f", gs.Pot)
	fmt.Println("---------")
	log.Printf("MineBet: %.1f", gs.MinBet)
	fmt.Println("---------")
	log.Printf("Phase: %s", gs.Phase)
	fmt.Println("--------------------------------")
}
