package gamestate

import "github.com/libp2p/go-libp2p/core/peer"

type GameState struct {
	Nicknames    map[peer.ID]string // Player nicknames tied to their peer.ID
	StartingCash float64
	TurnOrder    map[string]int     // Turn order of players by nickname
	PlayersMoney map[string]float64 // Players money by nickname
	WhosTurn     int                // Index of whos turn it is - used in tandem with Turn order
	Pot          float64            // The current pot amount
	MinBet       float64            // Minimum bet required for the round (again from table settings)
	Phase        string             // Current phase of the game (e.g., "preflop", "flop", "turn", "river")
	BetHistory   map[string]float64 // A map to store bets placed by players
}

// StartGame initializes the game state
func (gs *GameState) Init(numOfPlayers int, startingCash *float64, minBet *float64) {
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
