package p2p

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"goker/internal/channelmanager"
	"log"
	"math/big"
	"math/rand"
	"strings"

	"fyne.io/fyne/v2/canvas"
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
	CardValue      *big.Int
}

var ranks = [...]string{"ace", "2", "3", "4", "5", "6", "7", "8", "9", "10", "jack", "queen", "king"}
var suits = [...]string{"hearts", "diamonds", "clubs", "spades"}

// GenerateCardHash generates a hash for a card
func generateCardHash(card string, secretKey string) *big.Int {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(card))
	hBytes := h.Sum(nil)

	hash := new(big.Int).SetBytes(hBytes)
	return hash
}

// Creates new reference deck
func (d *deckInfo) GenerateDecks(key string) {
	newRefDeck := make(map[string]*big.Int, 52)
	newRoundDeck := make([]CardInfo, 0, 52)

	index := 0
	for _, suit := range suits {
		for _, rank := range ranks {
			cardName := suit + "_" + rank
			cardHash := generateCardHash(cardName, key)

			// we are setting a copy to the round and ref deck, so later we won't edit the ref deck on accident
			newRefDeck[cardName] = new(big.Int).Set(cardHash)
			newRoundDeck = append(newRoundDeck, CardInfo{index: index, CardValue: new(big.Int).Set(cardHash)})
			index++
		}
	}
	d.ReferenceDeck = newRefDeck
	d.RoundDeck = newRoundDeck
}

// Shuffle the round deck
func (d *deckInfo) ShuffleRoundDeck() {
	rand.Shuffle(len(d.RoundDeck), func(i, j int) {
		d.RoundDeck[i], d.RoundDeck[j] = d.RoundDeck[j], d.RoundDeck[i]
	})
}

// Returns the string of a given hash in the reference deck, ok determines if it's there or not
func (d *deckInfo) GetCardFromRefDeck(cardHash *big.Int) (key string, ok bool) {
	for k, v := range d.ReferenceDeck {
		if v.Cmp(cardHash) == 0 {
			return k, true
		}
	}
	return "", false
}

// Send a copy of the card we want
func (d *deckInfo) GetCardFromRoundDeck(cardIndex int) CardInfo {
	if cardIndex < 0 || cardIndex >= len(d.RoundDeck) {
		return CardInfo{}
	}

	return d.RoundDeck[cardIndex]
}

// Creates a payload to be sent to anther peer
func (d *deckInfo) GenerateDeckPayload() string {
	var lines []string

	for _, card := range d.RoundDeck {
		lines = append(lines, card.CardValue.String())
	}
	return strings.Join(lines, "\n")
}

// Used during first round of protocol, as variation order doesn't matter yet - i.e. the deck coming in is shuffled
func (d *deckInfo) SetNewDeck(payload string) {
	var newDeck []CardInfo
	lines := strings.Split(strings.TrimSpace(payload), "\n")
	for i, line := range lines {
		card, success := new(big.Int).SetString(line, 10)
		if !success {
			log.Printf("SetDeck: Failed to parse card value: %s", line)
			continue
		}
		newDeck = append(newDeck, CardInfo{index: i, CardValue: card})
	}
	d.RoundDeck = newDeck
}

// Just change the the decks card values, the order the cards come in correlate to the order of the deck
func (d *deckInfo) SetDeckInPlace(payload string) {
	lines := strings.Split(strings.TrimSpace(payload), "\n")

	if len(lines) != len(d.RoundDeck) {
		log.Printf("SetDeckInPlace: Mismatched deck sizes (RoundDeck: %d, Payload: %d)", len(d.RoundDeck), len(lines))
		return
	}

	for i, line := range lines {
		card, success := new(big.Int).SetString(line, 10)
		if !success {
			log.Printf("SetDeck: Failed to parse card value: %s", line)
			continue
		}
		d.RoundDeck[i].CardValue = card // Since we JUST want to change the card value
	}
}

// Decrypt round deck with global keys
func (p *GokerPeer) DecryptAllWithGlobalKeys() {
	for i := range p.Deck.RoundDeck {
		p.Keyring.DecryptWithGlobalKeys(p.Deck.RoundDeck[i].CardValue)
	}
}

// Encrypt round deck with global keys
func (p *GokerPeer) EncryptAllWithGlobalKeys() {
	for i := range p.Deck.RoundDeck {
		p.Keyring.EncryptWithGlobalKeys(p.Deck.RoundDeck[i].CardValue)
	}
}

// Encrypt deck with variation numbers
func (p *GokerPeer) EncryptAllWithVariation() {
	for i := range p.Deck.RoundDeck {
		err := p.Keyring.EncryptWithVariation(p.Deck.RoundDeck[i].CardValue, i)
		if err != nil {
			log.Println(err)
		}
		p.Deck.RoundDeck[i].VariationIndex = i
	}
}

// Gets the key payload for a players specific cards
func (p *GokerPeer) GetKeyPayloadForPlayersHand(peerID peer.ID) string {
	var keys []string

	IDs := p.gameState.GetTurnOrder()

	for i := range IDs {
		if IDs[i] == peerID { // If its the peer we want
			// Given the players cards variation index, get the corresponding key
			cardOneKey := p.Keyring.GetVariationKeyForCard(p.Deck.GetCardFromRoundDeck(i).VariationIndex)
			cardTwoKey := p.Keyring.GetVariationKeyForCard(p.Deck.GetCardFromRoundDeck(len(IDs) + i).VariationIndex)
			if cardOneKey == nil || cardTwoKey == nil {
				log.Fatalf("error: Could not retrieve key for one or both cards")
				continue
			}
			keys = append(keys, cardOneKey.String(), cardTwoKey.String())
			break
		}
	}

	if len(keys) != 2 {
		log.Println("warning: No matching hand found for peer")
	}

	fmt.Println("SENDING KEYS")
	fmt.Println(keys)

	return strings.Join(keys, "\n")
}

// Set my hand
func (p *GokerPeer) SetMyHand() {
	IDs := p.gameState.GetTurnOrder()

	for i, id := range IDs {
		if id == p.ThisHost.ID() { // put this host in another place
			cardOne := p.Deck.GetCardFromRoundDeck(i)
			cardTwo := p.Deck.GetCardFromRoundDeck(len(IDs) + i)

			if cardOne.CardValue == nil || cardTwo.CardValue == nil {
				log.Fatalf("error: Could not set my hand, missing cardOne or cardTwo\n")
			}

			p.MyHand = HandInfo{
				Peer: id,
				Hand: []CardInfo{cardOne, cardTwo},
			}
			break
		}
	}
}

// Decrypts my hand in the hands array given to key strings
func (p *GokerPeer) DecryptMyHand(cardOneKeys []string, cardTwoKeys []string) {
	// Decrypt card one
	for _, key := range cardOneKeys {
		keyOne, success := new(big.Int).SetString(key, 10)
		if !success {
			log.Println("DecryptMyHand: error: Unable to convert string to big.Int")
			return
		}
		//log.Printf("Decrypting Card 1: %s\n with Key: %s\n", p.MyHand.Hand[0].CardValue.String(), keyOne.String())
		p.Keyring.DecryptWithKey(p.MyHand.Hand[0].CardValue, keyOne)
	}

	// Decrypt card two
	for _, key := range cardTwoKeys {
		keyTwo, success := new(big.Int).SetString(key, 10)
		if !success {
			log.Println("DecryptMyHand: error: Unable to convert string to big.Int")
			return
		}
		p.Keyring.DecryptWithKey(p.MyHand.Hand[1].CardValue, keyTwo)
	}

	err := p.Keyring.DecryptWithVariation(p.MyHand.Hand[0].CardValue, p.MyHand.Hand[0].VariationIndex)
	if err != nil {
		return
	}

	err = p.Keyring.DecryptWithVariation(p.MyHand.Hand[1].CardValue, p.MyHand.Hand[1].VariationIndex)
	if err != nil {
		return
	}
}

// Load images for my hand and send them to GUI
func (p *GokerPeer) sendHandToGUI(cardOneName, cardTwoName string) {
	log.Printf("sendHandToGUI: Sending cards to GUI: %s, %s\n", cardOneName, cardTwoName)
	cardOne := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardOneName + ".png")
	cardOne.FillMode = canvas.ImageFillOriginal

	cardTwo := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardTwoName + ".png")
	cardTwo.FillMode = canvas.ImageFillOriginal

	newHand := make([]*canvas.Image, 0, 2)
	newHand = append(newHand, cardOne)
	newHand = append(newHand, cardTwo)
	channelmanager.TGUI_HandChan <- newHand
}

func (d *deckInfo) DumpRoundDeck() {
	for _, card := range d.RoundDeck {
		fmt.Println(card.CardValue)
	}
}
