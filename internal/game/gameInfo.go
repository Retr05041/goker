package game

import (
	"fmt"
	"math/big"
)

// Holds the current game info (updated per round)
type GameInfo struct {
	// Holds a map of name to hash
	ReferenceDeck       map[string]*big.Int
	// Holds hash's that will be encrypted and shuffled per round
	RoundDeck []Card

	// Secret for hashing cards - Needs to be shared
	deckHashSecret *big.Int
}

type Card struct {
	index int
	Cardvalue *big.Int
}

func (g *GameInfo) DisplayDeck() {
	for _, v := range g.RoundDeck {
		fmt.Print(v.index)
		fmt.Print(" - ")
		fmt.Println(v.Cardvalue.String())
	}
}
