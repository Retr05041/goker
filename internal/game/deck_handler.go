package game

import (
	"fmt"
	"math/big"
	"math/rand"
	"strconv"
)

var ranks = [...]string{"ACE", "TWO", "THREE", "FOUR", "FIVE", "SIX", "SEVEN", "EIGHT", "NINE", "TEN", "JACK", "QUEEN", "KING"}
var suits = [...]string{"HEARTS", "DIAMONDS", "CLUBS", "SPADES"}

// Creates new reference deck 
func (g *GameInfo) GenerateRefDeck(key string) {
	newRefDeck := make(map[string]*big.Int, 52)

	count := 0
	for _, suit := range suits {
		for _, rank := range ranks {
			cardName := rank + " " + suit
			cardHash := generateCardHash(cardName, key)

			newRefDeck[cardName] = cardHash
			fmt.Println(strconv.Itoa(count) + " - " + cardName + " - " + cardHash.String())
			count++
		}
	}
	g.ReferenceDeck = newRefDeck
}

// Generate the round deck, which will just be all the hash's from the reference deck
func (g *GameInfo) GenerateRoundDeck() {
	newDeck := make([]*big.Int, 0, 52)
	for _, hash := range g.ReferenceDeck {
		newDeck = append(newDeck, hash)
	}
	g.RoundDeck = newDeck
}

// Shuffle the round deck
func (g *GameInfo) ShuffleRoundDeck() {
	rand.Shuffle(len(g.RoundDeck), func(i, j int) {
		g.RoundDeck[i], g.RoundDeck[j] = g.RoundDeck[j], g.RoundDeck[i]
	})
}

func TestDeck() {
	newGame := new(GameInfo)

	newGame.GenerateRefDeck("mysupersecretkey")
	newGame.GenerateRoundDeck()
	for i, v := range newGame.RoundDeck {
		fmt.Print(strconv.Itoa(i) + " - ")
		fmt.Println(v.String())
	}

	fmt.Println("---")

	newGame.ShuffleRoundDeck()
	for i, v := range newGame.RoundDeck {
		fmt.Print(strconv.Itoa(i) + " - ")
		fmt.Println(v.String())
	}

}