package game

import (
	"fmt"
	"math/big"
	"math/rand"
	"strings"
)

var ranks = [...]string{"ace", "2", "3", "4", "5", "6", "7", "8", "9", "10", "jack", "queen", "king"}
var suits = [...]string{"hearts", "diamonds", "clubs", "spades"}

// Creates new reference deck 
func (g *GameInfo) GenerateRefDeck(key string) {
	newRefDeck := make(map[string]*big.Int, 52)

	count := 0
	for _, suit := range suits {
		for _, rank := range ranks {
			cardName := suit + "_" + rank
			cardHash := generateCardHash(cardName, key)

			newRefDeck[cardName] = cardHash
			count++
		}
	}
	g.ReferenceDeck = newRefDeck
}

// Generate the round deck, which will just be all the hash's from the reference deck
func (g *GameInfo) GenerateRoundDeck(key string) {
	newDeck := make([]Card, 0, 52)

	index := 0
	for _, suit := range suits {
		for _, rank := range ranks {
			cardName := suit + "_" + rank
			cardHash := generateCardHash(cardName, key)
			newDeck = append(newDeck, Card{index: index, Cardvalue: cardHash})
			index++
		}
	}
	g.RoundDeck = newDeck
}

// Shuffle the round deck
func (g *GameInfo) ShuffleRoundDeck() {
	rand.Shuffle(len(g.RoundDeck), func(i, j int) {
		g.RoundDeck[i], g.RoundDeck[j] = g.RoundDeck[j], g.RoundDeck[i]
	})
}

func (g *GameInfo) GetCardFromRefDeck(cardHash *big.Int) (key string, ok bool) {
  for k, v := range g.ReferenceDeck {
    if v.Cmp(cardHash) == 0 { 
      return k, true
    }
  }
  return "", false
}

// Creates a payload to be sent to anther peer
func (g *GameInfo) GenerateDeckPayload() string {
	// Start with global public and private keys
	payload := ""

	// Append each variation's `r` value
	for _, card := range g.RoundDeck {
		payload += fmt.Sprintf("%s\n", card.Cardvalue.String())
	}

	return payload
}

// Set the round deck given a payload from another peer in the network
func (g *GameInfo) SetDeck(payload string) {
	var newDeck []Card
	for i, line := range strings.Split(payload, "\n") {
		if line == "" || line == "\\END" {
			continue
		}
		card, _ := new(big.Int).SetString(line, 10)
		newDeck = append(newDeck, Card{index: i, Cardvalue: card})
	}
	g.RoundDeck = newDeck
}
