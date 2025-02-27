package p2p

import (
	"math/big"
	"strings"
	"testing"
)

func TestShuffleRoundDeck(t *testing.T) {
	deck := &deckInfo{}
	deck.GenerateDecks("testkey")
	originalOrder := make([]big.Int, len(deck.RoundDeck))

	for i, card := range deck.RoundDeck {
		originalOrder[i] = *card.CardValue
	}

	deck.ShuffleRoundDeck()
	different := false
	for i, card := range deck.RoundDeck {
		if originalOrder[i].Cmp(card.CardValue) != 0 {
			different = true
			break
		}
	}

	if !different {
		t.Errorf("ShuffleRoundDeck did not change the order of cards")
	}
}

func TestGenerateDeckPayload(t *testing.T) {
	deck := &deckInfo{}
	deck.GenerateDecks("testkey")
	payload := deck.GenerateDeckPayload()

	if len(strings.Split(payload, "\n")) != 52 {
		t.Errorf("Expected payload with 52 lines, got %d", len(strings.Split(payload, "\n")))
	}
}
