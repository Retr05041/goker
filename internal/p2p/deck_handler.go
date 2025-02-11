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

func (d *deckInfo) DisplayDeck() {
	for i, v := range d.RoundDeck {
		fmt.Print(i)
		fmt.Print(" - ")
		fmt.Println(v.Cardvalue.String())
	}
}

// Creates new reference deck
func (d *deckInfo) GenerateRefDeck(key string) {
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
	d.ReferenceDeck = newRefDeck
}

// Generate the round deck, which will just be all the hash's from the reference deck
func (d *deckInfo) GenerateRoundDeck(key string) {
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
	d.RoundDeck = newDeck
}

// Shuffle the round deck
func (d *deckInfo) ShuffleRoundDeck() {
	rand.Shuffle(len(d.RoundDeck), func(i, j int) {
		d.RoundDeck[i], d.RoundDeck[j] = d.RoundDeck[j], d.RoundDeck[i]
	})
}

func (d *deckInfo) GetCardFromRefDeck(cardHash *big.Int) (key string, ok bool) {
	for k, v := range d.ReferenceDeck {
		if v.Cmp(cardHash) == 0 {
			return k, true
		}
	}
	return "", false
}

func (d *deckInfo) GetCardFromRoundDeck(cardIndex int) *CardInfo {
	for i, v := range d.RoundDeck {
		if i == cardIndex {
			return &v
		}
	}
	return nil
}

// Creates a payload to be sent to anther peer
func (d *deckInfo) GenerateDeckPayload() string {
	// Start with global public and private keys
	payload := ""

	// Append each variation's `r` value
	for _, card := range d.RoundDeck {
		payload += fmt.Sprintf("%s\n", card.Cardvalue.String())
	}

	return payload
}

// Set the round deck given a payload from another peer in the network
func (d *deckInfo) SetDeck(payload string) {
	var newDeck []CardInfo
	for i, line := range strings.Split(payload, "\n") {
		if line == "" || line == "\\END" {
			continue
		}
		card, _ := new(big.Int).SetString(line, 10)
		newDeck = append(newDeck, CardInfo{index: i, Cardvalue: card})
	}
	d.RoundDeck = newDeck
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

// Gets the key payload for a players specific cards
func (p *GokerPeer) GetKeyPayloadForPlayersHand(peerID peer.ID) string {
	payload := ""

	for _, handInfo := range p.Hands {
		if handInfo.Peer == peerID {
			if len(handInfo.Hand) < 2 {
				log.Println("error: Hand has less than 2 cards for peer")
				continue
			}

			cardOneKey := p.Keyring.GetKeyForCard(handInfo.Hand[0].VariationIndex)
			cardTwoKey := p.Keyring.GetKeyForCard(handInfo.Hand[1].VariationIndex)

			if cardOneKey == nil || cardTwoKey == nil {
				log.Println("error: Could not retrieve key for one or both cards")
				continue
			}

			payload += fmt.Sprintf("%s\n%s\n", cardOneKey.String(), cardTwoKey.String())
			break
		}
	}

	if payload == "" {
		log.Println("warning: No matching hand found for peer")
	}

	return payload
}

// Creates the hands array, Should be done once the second step of the protocol is done
func (p *GokerPeer) SetHands() {
	IDs := p.gameState.GetTurnOrder()

	p.Hands = make([]HandInfo, 0, len(IDs))

	for i, id := range IDs {
		cardOne := p.Deck.GetCardFromRoundDeck(i)
		cardTwo := p.Deck.GetCardFromRoundDeck(len(IDs) + i)
		p.Hands = append(p.Hands, HandInfo{
			Peer: id,
			Hand: []CardInfo{*cardOne, *cardTwo},
		})
	}
}

// Decrypts my hand in the hands array given to key strings
func (p *GokerPeer) DecryptMyHand(cardOneKey string, cardTwoKey string) {}
