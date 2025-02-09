package p2p

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
)

// Holds the current game info (updated per round)
type deckInfo struct {
	// Holds a map of name to hash
	ReferenceDeck map[string]*big.Int
	// Holds hash's that will be encrypted and shuffled per round
	RoundDeck []CardInfo

	// Secret for hashing cards - Needs to be shared
	deckHashSecret *big.Int
}

// Holds a peers hand info
type HandInfo struct {
	Peer peer.ID
	Hand []CardInfo	
}

// Holds individual card info
type CardInfo struct {
	index          int
	VariationIndex int
	Cardvalue      *big.Int
}

var ranks = [...]string{"ace", "2", "3", "4", "5", "6", "7", "8", "9", "10", "jack", "queen", "king"}
var suits = [...]string{"hearts", "diamonds", "clubs", "spades"}

// GenerateCardHash generates a hash for a card
func generateCardHash(card string, secretKey string) *big.Int {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(card))
	hBytes := h.Sum(nil)
	return new(big.Int).SetBytes(hBytes)
}

// VerifyCardHash verifies if the hash matches the card
func VerifyCardHash(card string, secretKey string, hash *big.Int) bool {
	expectedHash := generateCardHash(card, secretKey)
	return expectedHash.Cmp(hash) == 0
}



func (g *deckInfo) DisplayDeck() {
	for i, v := range g.RoundDeck {
		fmt.Print(i)
		fmt.Print(" - ")
		fmt.Println(v.Cardvalue.String())
	}
}

// Creates new reference deck 
func (g *deckInfo) GenerateRefDeck(key string) {
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
func (g *deckInfo) GenerateRoundDeck(key string) {
	newDeck := make([]CardInfo, 0, 52)

	index := 0
	for _, suit := range suits {
		for _, rank := range ranks {
			cardName := suit + "_" + rank
			cardHash := generateCardHash(cardName, key)
			newDeck = append(newDeck, CardInfo{index: index, Cardvalue: cardHash})
			index++
		}
	}
	g.RoundDeck = newDeck
}

// Shuffle the round deck
func (g *deckInfo) ShuffleRoundDeck() {
	rand.Shuffle(len(g.RoundDeck), func(i, j int) {
		g.RoundDeck[i], g.RoundDeck[j] = g.RoundDeck[j], g.RoundDeck[i]
	})
}

func (g *deckInfo) GetCardFromRefDeck(cardHash *big.Int) (key string, ok bool) {
  for k, v := range g.ReferenceDeck {
    if v.Cmp(cardHash) == 0 { 
      return k, true
    }
  }
  return "", false
}

// Creates a payload to be sent to anther peer
func (g *deckInfo) GenerateDeckPayload() string {
	// Start with global public and private keys
	payload := ""

	// Append each variation's `r` value
	for _, card := range g.RoundDeck {
		payload += fmt.Sprintf("%s\n", card.Cardvalue.String())
	}

	return payload
}

// Set the round deck given a payload from another peer in the network
func (g *deckInfo) SetDeck(payload string) {
	var newDeck []CardInfo
	for i, line := range strings.Split(payload, "\n") {
		if line == "" || line == "\\END" {
			continue
		}
		card, _ := new(big.Int).SetString(line, 10)
		newDeck = append(newDeck, CardInfo{index: i, Cardvalue: card})
	}
	g.RoundDeck = newDeck
}


// Decrypt deck with global keys
func (p *GokerPeer) DecryptAllWithGlobalKeys() {
	for _, card := range p.Deck.RoundDeck {
		card.Cardvalue = p.Keyring.DecryptWithGlobalKeys(card.Cardvalue)
	}
}

// Encrypt deck with global keys
func (p *GokerPeer) EncryptAllWithGlobalKeys() {
	for _, card := range p.Deck.RoundDeck {
		card.Cardvalue = p.Keyring.EncryptWithGlobalKeys(card.Cardvalue)
	}
}

// Encrypt deck with variation numbers
func (p *GokerPeer) EncryptAllWithVariation() {
	for i, card := range p.Deck.RoundDeck {
		encryptedCard, err := p.Keyring.EncryptWithVariation(card.Cardvalue, i)
		if err != nil {
			log.Println(err)
		}
		p.Deck.RoundDeck[i].Cardvalue = encryptedCard
		p.Deck.RoundDeck[i].VariationIndex = i
	}
}
