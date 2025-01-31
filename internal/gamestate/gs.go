package gamestate

import (
	"log"

	"github.com/libp2p/go-libp2p/core/peer"
)

type GameState struct {
	Players      map[peer.ID]string // Player nicknames tied to their peer.ID
	PlayersMoney map[string]float64 // Players money by nickname
	WhosTurn     peer.ID

	StartingCash float64
	Pot          float64            // The current pot amount
	MinBet       float64            // Minimum bet required for the round (again from table settings)
	Phase        string             // Current phase of the game (e.g., "preflop", "flop", "turn", "river")
	BetHistory   map[string]float64 // A map to store bets placed by players
}

// Refresh state for new possible rounds
func (gs *GameState) FreshState(startingCash *float64, minBet *float64) {
	gs.Phase = "preflop"
	gs.Pot = 0.0

	gs.StartingCash = 100.0
	if startingCash != nil {
		gs.StartingCash = *startingCash
	}

	if minBet == nil {
		gs.MinBet = 1.0
	} else {
		gs.MinBet = *minBet
	}
}

// For adding a new peer to the state
func (gs *GameState) AddPeerToState(peerID peer.ID, nickname string, turnIndex int) {
	if _, v := gs.Players[peerID]; v {
		log.Println("AddPeerToState: Peer already in state")
		return
	}
	gs.Players[peerID] = nickname
	gs.PlayersMoney[nickname] = gs.StartingCash
}

func (gs *GameState) RemovePeerFromState(peerID peer.ID) {
	nickname, exists := gs.Players[peerID]

	if !exists {
		log.Println("RemovePeerFromState: Peer not in state")
		return
	}

	delete(gs.Players, peerID)
	delete(gs.PlayersMoney, nickname)
	delete(gs.BetHistory, nickname)
}

func (gs *GameState) GetCurrentPot() float64 {
	pot := 0.0
	for _, b := range gs.BetHistory {
		pot += b
	}
	return pot
}
