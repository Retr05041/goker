package gamestate

import (
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/peer"
)

type GameState struct {
	// Handled as players join
	Players      map[peer.ID]string  // Player nicknames tied to their peer.ID
	PlayersMoney map[peer.ID]float64 // Players money by peer ID
	BetHistory   map[peer.ID]float64 // A map to store bets placed on the current round

	// Handled by network (based off of candidate list)
	TurnOrder map[int]peer.ID
	WhosTurn  int

	// Set by host and shared in network
	StartingCash float64
	Pot          float64 // The current pot amount
	MinBet       float64 // Minimum bet required for the round (again from table settings)
	Phase        string  // Current phase of the game (e.g., "preflop", "flop", "turn", "river")
}

// Refresh state for new possible rounds
func (gs *GameState) FreshState(startingCash *float64, minBet *float64) {
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
	if _, v := gs.Players[peerID]; v {
		log.Println("AddPeerToState: Peer already in state")
		return
	}
	gs.Players[peerID] = nickname
	gs.BetHistory[peerID] = 0.0
}

func (gs *GameState) RemovePeerFromState(peerID peer.ID) {
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

func (gs *GameState) GetCurrentPot() float64 {
	pot := 0.0
	for _, b := range gs.BetHistory {
		pot += b
	}
	return pot
}

func (gs *GameState) SetTurnOrder(IDs []peer.ID) {
	for i, v := range IDs {
		gs.TurnOrder[i] = v
	}
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
