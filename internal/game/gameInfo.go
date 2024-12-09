package game

import (
	"math/big"
)

type GameInfo struct {
	// Deck handler
	deck       map[string]*big.Int
	deckSecret *big.Int
}
