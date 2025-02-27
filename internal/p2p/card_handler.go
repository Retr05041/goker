package p2p

import (
	"crypto/hmac"
	"crypto/sha256"
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

// VerifyCardHash verifies if the hash matches the card
func VerifyCardHash(card string, secretKey string, hash *big.Int) bool {
	expectedHash := generateCardHash(card, secretKey)
	return expectedHash.Cmp(hash) == 0
}

// Creates new reference deck
func (d *deckInfo) GenerateRefDeck(key string) {
	newRefDeck := make(map[string]*big.Int, 52)

	for _, suit := range suits {
		for _, rank := range ranks {
			cardName := suit + "_" + rank
			cardHash := generateCardHash(cardName, key)

			newRefDeck[cardName] = cardHash
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
			newDeck = append(newDeck, CardInfo{index: index, CardValue: cardHash})
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

// Send a copy of the card at this index
func (d *deckInfo) GetCardFromRoundDeck(cardIndex int) *CardInfo {
	if cardIndex < 0 || cardIndex >= len(d.RoundDeck) {
		return nil
	}

	cardCopy := d.RoundDeck[cardIndex]
	return &cardCopy
}

// Creates a payload to be sent to anther peer
func (d *deckInfo) GenerateDeckPayload() string {
	var lines []string

	for _, card := range d.RoundDeck {
		lines = append(lines, card.CardValue.String())
	}
	return strings.Join(lines, "\n")
}

// Used during first round of protocol, as variation order doesn't mater
func (d *deckInfo) SetNewDeck(payload string) {
	var newDeck []CardInfo
	for i, line := range strings.Split(payload, "\n") {
		card, success := new(big.Int).SetString(line, 10)
		if !success {
			log.Printf("SetDeck: Failed to parse card value: %s", line)
			continue
		}
		newDeck = append(newDeck, CardInfo{index: i, CardValue: card})
	}
	d.RoundDeck = newDeck
}

func (d *deckInfo) SetDeckInPlace(payload string) {
	for i, line := range strings.Split(payload, "\n") {
		card, success := new(big.Int).SetString(line, 10)
		if !success {
			log.Printf("SetDeck: Failed to parse card value: %s", line)
			continue
		}
		d.RoundDeck[i].CardValue = card // Since we JUST want to change the card value
	}
}

// Decrypt deck with global keys
func (p *GokerPeer) DecryptAllWithGlobalKeys() {
	for i := range p.Deck.RoundDeck {
		p.Deck.RoundDeck[i].CardValue = p.Keyring.DecryptWithGlobalKeys(p.Deck.RoundDeck[i].CardValue)
	}
}

// Encrypt deck with global keys
func (p *GokerPeer) EncryptAllWithGlobalKeys() {
	for i := range p.Deck.RoundDeck {
		p.Deck.RoundDeck[i].CardValue = p.Keyring.EncryptWithGlobalKeys(p.Deck.RoundDeck[i].CardValue)
	}
}

// Encrypt deck with variation numbers
func (p *GokerPeer) EncryptAllWithVariation() {
	for i, card := range p.Deck.RoundDeck {
		encryptedCard, err := p.Keyring.EncryptWithVariation(card.CardValue, i)
		if err != nil {
			log.Println(err)
		}
		p.Deck.RoundDeck[i].CardValue = encryptedCard
		p.Deck.RoundDeck[i].VariationIndex = i
	}
}

// Gets the key payload for a players specific cards
func (p *GokerPeer) GetKeyPayloadForPlayersHand(peerID peer.ID) string {
	var keys []string

	IDs := p.gameState.GetTurnOrder()

	for i := range IDs {
		if IDs[i] == peerID { // If its the peer we want
			cardOneKey := p.Keyring.GetKeyForCard(p.Deck.GetCardFromRoundDeck(i).VariationIndex)
			cardTwoKey := p.Keyring.GetKeyForCard(p.Deck.GetCardFromRoundDeck(len(IDs) + i).VariationIndex)
			if cardOneKey == nil || cardTwoKey == nil {
				log.Fatalf("error: Could not retrieve key for one or both cards")
				continue
			}
			keys = append(keys, cardOneKey.String())
			keys = append(keys, cardTwoKey.String())
			break
		}
	}

	if len(keys) != 2 {
		log.Println("warning: No matching hand found for peer")
	}

	return strings.Join(keys, "\n")
}

// Set my hand
func (p *GokerPeer) SetMyHand() {
	IDs := p.gameState.GetTurnOrder()

	for i, id := range IDs {
		if id == p.ThisHost.ID() { // put this host in another place
			cardOne := p.Deck.GetCardFromRoundDeck(i)
			cardTwo := p.Deck.GetCardFromRoundDeck(len(IDs) + i)
			p.MyHand = HandInfo{
				Peer: id,
				Hand: []CardInfo{*cardOne, *cardTwo},
			}
			break
		}
	}
}

// Decrypts my hand in the hands array given to key strings
func (p *GokerPeer) DecryptMyHand(cardOneKeys []string, cardTwoKeys []string) {
	for _, key := range cardOneKeys {
		keyOne, success := new(big.Int).SetString(key, 10)
		if !success {
			log.Println("DecryptMyHand: error: Unable to convert string to big.Int")
			return
		}
		//log.Printf("Decrypting Card 1: %s\n with Key: %s\n", p.MyHand.Hand[0].CardValue.String(), keyOne.String())
		myCardOne := p.Keyring.DecryptWithKey(p.MyHand.Hand[0].CardValue, keyOne)
		p.MyHand.Hand[0].CardValue = myCardOne
	}

	for _, key := range cardTwoKeys {
		keyTwo, success := new(big.Int).SetString(key, 10)
		if !success {
			log.Println("DecryptMyHand: error: Unable to convert string to big.Int")
			return
		}
		//log.Printf("Decrypting Card 2: %s\n with Key: %s\n", p.MyHand.Hand[1].CardValue.String(), keyTwo.String())
		myCardtwo := p.Keyring.DecryptWithKey(p.MyHand.Hand[1].CardValue, keyTwo)
		p.MyHand.Hand[1].CardValue = myCardtwo
	}

	plaintextCardOne, err := p.Keyring.DecryptWithVariation(p.MyHand.Hand[0].CardValue, p.MyHand.Hand[0].VariationIndex)
	if err != nil {
		return
	}
	p.MyHand.Hand[0].CardValue = plaintextCardOne

	plaintextCardTwo, err := p.Keyring.DecryptWithVariation(p.MyHand.Hand[1].CardValue, p.MyHand.Hand[1].VariationIndex)
	if err != nil {
		return
	}
	p.MyHand.Hand[1].CardValue = plaintextCardTwo
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
