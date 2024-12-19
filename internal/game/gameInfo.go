package game

import (
	"math/big"
)

// Holds the current game info (updated per round)
type GameInfo struct {
	// Holds a map of name to hash
	ReferenceDeck       map[string]*big.Int
	// Holds hash's that will be encrypted and shuffled per round
	RoundDeck []*big.Int

	// Secret for hashing cards - Needs to be shared
	deckHashSecret *big.Int
}
