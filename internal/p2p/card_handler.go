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

// Holds individual card info
type CardInfo struct {
	index          int
	VariationIndex int
	CardValue      *big.Int
	CardKeys       []string // This will hold the keys used to decrypt this card..
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
	if !d.verifyUniqueHashes() {
		log.Fatalf("Did not generate a good deck.")
	}
}

func (d *deckInfo) verifyUniqueHashes() bool {
	hashSet := make(map[string]bool, len(d.ReferenceDeck))

	for cardName, cardHash := range d.ReferenceDeck {
		hashStr := cardHash.String()
		if hashSet[hashStr] {
			log.Printf("Duplicate hash found for card: %s (hash: %s)\n", cardName, hashStr)
			return false
		}
		hashSet[hashStr] = true
	}
	return true
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

// Get a pointer to the card in the round deck at a given index
func (d *deckInfo) GetCardFromRoundDeck(cardIndex int) *CardInfo {
	if cardIndex < 0 || cardIndex >= len(d.RoundDeck) {
		return &CardInfo{}
	}

	return &d.RoundDeck[cardIndex]
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

	return strings.Join(keys, "\n")
}

func (p *GokerPeer) SetHands() {
	IDs := p.gameState.GetTurnOrder()

	for i, id := range IDs {
		cardOne := p.Deck.GetCardFromRoundDeck(i)
		cardTwo := p.Deck.GetCardFromRoundDeck(len(IDs) + i)
		if cardOne.CardValue == nil || cardTwo.CardValue == nil {
			log.Fatalf("error: Could not set my hand, missing cardOne or cardTwo\n")
		}
		if id == p.ThisHost.ID() { // put this host in another place
			p.MyHand = []*CardInfo{cardOne, cardTwo}
		} else {
			p.OthersHands[id] = []*CardInfo{cardOne, cardTwo}
		}
	}
}

// Decrypts my hand in the hands array given to key strings
func (p *GokerPeer) DecryptMyHand(cardOneKeys []string, cardTwoKeys []string) {
	myKeyOne := p.Keyring.GetVariationKeyForCard(p.MyHand[0].VariationIndex)
	myKeyTwo := p.Keyring.GetVariationKeyForCard(p.MyHand[1].VariationIndex)

	cardOneKeys = append(cardOneKeys, myKeyOne.String())
	cardTwoKeys = append(cardTwoKeys, myKeyTwo.String())

	// Decrypt card one
	for _, key := range cardOneKeys {
		keyOne, success := new(big.Int).SetString(key, 10)
		if !success {
			log.Println("DecryptMyHand: error: Unable to convert string to big.Int")
			return
		}
		p.Keyring.DecryptWithKey(p.MyHand[0].CardValue, keyOne)
	}
	p.MyHand[0].CardKeys = cardOneKeys // Save the keys for later

	// Decrypt card two
	for _, key := range cardTwoKeys {
		keyTwo, success := new(big.Int).SetString(key, 10)
		if !success {
			log.Println("DecryptMyHand: error: Unable to convert string to big.Int")
			return
		}
		p.Keyring.DecryptWithKey(p.MyHand[1].CardValue, keyTwo)
	}
	p.MyHand[1].CardKeys = cardTwoKeys // Save the keys for later
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

func (p *GokerPeer) SetBoard() {
	numOfPlayers := p.gameState.GetNumberOfPlayers()

	cardOne := p.Deck.GetCardFromRoundDeck((numOfPlayers * 2) + 1) // all players hands + burn + first card
	cardTwo := p.Deck.GetCardFromRoundDeck((numOfPlayers * 2) + 2)
	cardThree := p.Deck.GetCardFromRoundDeck((numOfPlayers * 2) + 3)

	cardFour := p.Deck.GetCardFromRoundDeck((numOfPlayers * 2) + 4)

	cardFive := p.Deck.GetCardFromRoundDeck((numOfPlayers * 2) + 5)

	if cardOne.CardValue == nil || cardTwo.CardValue == nil || cardThree.CardValue == nil || cardFour.CardValue == nil || cardFive.CardValue == nil {
		log.Fatalf("error: Could not set board, missing cards \n")
	}

	p.Flop = []*CardInfo{cardOne, cardTwo, cardThree}
	p.Turn = cardFour
	p.River = cardFive
}

func (p *GokerPeer) DecryptFlop(cardOneKeys, cardTwoKeys, cardThreeKeys []string) {
	// Decrypt card one
	for _, key := range cardOneKeys {
		if p.gameState.Contains(p.Flop[0].CardKeys, key) {
			continue
		}

		keyOne, success := new(big.Int).SetString(key, 10)
		if !success {
			log.Println("DecryptFlop: error: Unable to convert string to big.Int")
			return
		}
		p.Keyring.DecryptWithKey(p.Flop[0].CardValue, keyOne)
		p.Flop[2].CardKeys = append(p.Flop[0].CardKeys, key)
	}

	// Decrypt card two
	for _, key := range cardTwoKeys {
		if p.gameState.Contains(p.Flop[1].CardKeys, key) {
			continue
		}

		keyTwo, success := new(big.Int).SetString(key, 10)
		if !success {
			log.Println("DecryptFlop: error: Unable to convert string to big.Int")
			return
		}
		p.Keyring.DecryptWithKey(p.Flop[1].CardValue, keyTwo)
		p.Flop[2].CardKeys = append(p.Flop[1].CardKeys, key)
	}

	// Decrypt card three
	for _, key := range cardThreeKeys {
		if p.gameState.Contains(p.Flop[2].CardKeys, key) {
			continue
		}

		keyThree, success := new(big.Int).SetString(key, 10)
		if !success {
			log.Println("DecryptFlop: error: Unable to convert string to big.Int")
			return
		}
		p.Keyring.DecryptWithKey(p.Flop[2].CardValue, keyThree)
		p.Flop[2].CardKeys = append(p.Flop[2].CardKeys, key)
	}

	err := p.Keyring.DecryptWithVariation(p.Flop[0].CardValue, p.Flop[0].VariationIndex)
	if err != nil {
		return
	}

	err = p.Keyring.DecryptWithVariation(p.Flop[1].CardValue, p.Flop[1].VariationIndex)
	if err != nil {
		return
	}

	err = p.Keyring.DecryptWithVariation(p.Flop[2].CardValue, p.Flop[2].VariationIndex)
	if err != nil {
		return
	}
}

// Update board for GUI
func (p *GokerPeer) sendBoardToGUI(cardOneName, cardTwoName, cardThreeName, cardFourName, cardFiveName *string) {
	var cardOne *canvas.Image
	if cardOneName != nil {
		cardOne = canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + *cardOneName + ".png")
		cardOne.FillMode = canvas.ImageFillOriginal
	} else {
		cardOne = canvas.NewImageFromFile("media/svg_playing_cards/backs/png_96_dpi/red.png")
		cardOne.FillMode = canvas.ImageFillOriginal
	}

	var cardTwo *canvas.Image
	if cardTwoName != nil {
		cardTwo = canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + *cardTwoName + ".png")
		cardTwo.FillMode = canvas.ImageFillOriginal
	} else {
		cardTwo = canvas.NewImageFromFile("media/svg_playing_cards/backs/png_96_dpi/red.png")
		cardTwo.FillMode = canvas.ImageFillOriginal
	}

	var cardThree *canvas.Image
	if cardThreeName != nil {
		cardThree = canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + *cardThreeName + ".png")
		cardThree.FillMode = canvas.ImageFillOriginal
	} else {
		cardThree = canvas.NewImageFromFile("media/svg_playing_cards/backs/png_96_dpi/red.png")
		cardThree.FillMode = canvas.ImageFillOriginal
	}

	var cardFour *canvas.Image
	if cardFourName != nil {
		cardFour = canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + *cardFourName + ".png")
		cardFour.FillMode = canvas.ImageFillOriginal
	} else {
		cardFour = canvas.NewImageFromFile("media/svg_playing_cards/backs/png_96_dpi/red.png")
		cardFour.FillMode = canvas.ImageFillOriginal
	}

	var cardFive *canvas.Image
	if cardFiveName != nil {
		cardFive = canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + *cardFiveName + ".png")
		cardFive.FillMode = canvas.ImageFillOriginal
	} else {
		cardFive = canvas.NewImageFromFile("media/svg_playing_cards/backs/png_96_dpi/red.png")
		cardFive.FillMode = canvas.ImageFillOriginal
	}

	newBoard := make([]*canvas.Image, 0, 5)
	newBoard = append(newBoard, cardOne, cardTwo, cardThree, cardFour, cardFive)
	channelmanager.TGUI_BoardChan <- newBoard
}

func (p *GokerPeer) GetKeyPayloadForFlop() string {
	var keys []string

	numOfPlayers := p.gameState.GetNumberOfPlayers()

	cardOneKey := p.Keyring.GetVariationKeyForCard(p.Deck.GetCardFromRoundDeck((numOfPlayers * 2) + 1).VariationIndex)
	cardTwoKey := p.Keyring.GetVariationKeyForCard(p.Deck.GetCardFromRoundDeck((numOfPlayers * 2) + 2).VariationIndex)
	cardThreeKey := p.Keyring.GetVariationKeyForCard(p.Deck.GetCardFromRoundDeck((numOfPlayers * 2) + 3).VariationIndex)

	if cardOneKey == nil || cardTwoKey == nil || cardThreeKey == nil {
		log.Fatalf("error: Could not retrieve key for cards")
	}
	keys = append(keys, cardOneKey.String(), cardTwoKey.String(), cardThreeKey.String())

	if len(keys) != 3 {
		log.Println("warning: No matching hand found for peer")
	}

	return strings.Join(keys, "\n")
}

func (p *GokerPeer) GetKeyPayloadForTurn() string {
	numOfPlayers := p.gameState.GetNumberOfPlayers()

	turnKey := p.Keyring.GetVariationKeyForCard(p.Deck.GetCardFromRoundDeck((numOfPlayers * 2) + 4).VariationIndex)

	if turnKey == nil {
		log.Fatalf("error: Could not retrieve key for cards")
	}

	return turnKey.String()
}

func (p *GokerPeer) GetKeyPayloadForRiver() string {
	numOfPlayers := p.gameState.GetNumberOfPlayers()

	riverKey := p.Keyring.GetVariationKeyForCard(p.Deck.GetCardFromRoundDeck((numOfPlayers * 2) + 5).VariationIndex)

	if riverKey == nil {
		log.Fatalf("error: Could not retrieve key for cards")
	}

	return riverKey.String()
}

func (p *GokerPeer) DecryptTurn(turnKeys []string) {
	for _, key := range turnKeys {
		if p.gameState.Contains(p.Turn.CardKeys, key) {
			continue
		}
		turnKey, success := new(big.Int).SetString(key, 10)
		if !success {
			log.Println("DecryptTurn: error: Unable to convert string to big.Int")
			return
		}
		p.Keyring.DecryptWithKey(p.Turn.CardValue, turnKey)
		p.Turn.CardKeys = append(p.Turn.CardKeys, key)
	}

	err := p.Keyring.DecryptWithVariation(p.Turn.CardValue, p.Turn.VariationIndex)
	if err != nil {
		return
	}
}

func (p *GokerPeer) DecryptRiver(riverKeys []string) {
	for _, key := range riverKeys {
		if p.gameState.Contains(p.River.CardKeys, key) {
			continue
		}
		riverKey, success := new(big.Int).SetString(key, 10)
		if !success {
			log.Println("DecryptRiver: error: Unable to convert string to big.Int")
			return
		}
		p.Keyring.DecryptWithKey(p.River.CardValue, riverKey)
		p.River.CardKeys = append(p.River.CardKeys, key)
	}

	err := p.Keyring.DecryptWithVariation(p.River.CardValue, p.River.VariationIndex)
	if err != nil {
		return
	}
}

func (p *GokerPeer) DecryptOthersHand(peerID peer.ID, keys []string) {
	// Ensure the peer exists in OthersHands
	if _, exists := p.OthersHands[peerID]; !exists {
		fmt.Println("Peer not found in OthersHands")
		return
	}

	// Split keys into two sets (first half for card one, second half for card two)
	n := len(keys) / 2
	if len(keys)%2 != 0 || n == 0 {
		fmt.Println("Invalid number of keys provided")
		return
	}

	cardOneKeys := keys[:n]
	cardTwoKeys := keys[n:]

	// Decrypt card one
	for _, keyStr := range cardOneKeys {
		if p.gameState.Contains(p.OthersHands[peerID][0].CardKeys, keyStr) { // Skip keys already used
			continue
		}

		key, success := new(big.Int).SetString(keyStr, 10)
		if !success {
			log.Println("DecryptOthersHand: error: Unable to convert string to big.Int")
			return
		}

		p.Keyring.DecryptWithKey(p.OthersHands[peerID][0].CardValue, key)
	}
	p.OthersHands[peerID][0].CardKeys = cardOneKeys

	// Decrypt card two
	for _, keyStr := range cardTwoKeys {
		if p.gameState.Contains(p.OthersHands[peerID][1].CardKeys, keyStr) {
			continue
		}

		key, success := new(big.Int).SetString(keyStr, 10)
		if !success {
			log.Println("DecryptOthersHand: error: Unable to convert string to big.Int")
			return
		}
		p.Keyring.DecryptWithKey(p.OthersHands[peerID][1].CardValue, key)
	}
	p.OthersHands[peerID][1].CardKeys = cardTwoKeys
}

func (p *GokerPeer) GetKeyPayloadForMyHand() string {
	var allKeys []string

	if p.MyHand[0].CardKeys == nil || p.MyHand[1].CardKeys == nil {
		log.Fatalf("error: Could not retrieve keys for cards")
	}

	allKeys = append(allKeys, p.MyHand[0].CardKeys...)
	allKeys = append(allKeys, p.MyHand[1].CardKeys...)

	return strings.Join(allKeys, "\n")
}

// Given a keyring payload, generated in variations.go, we can decrypt the round deck
// (if a given key has yet to be used on that card)
func (p *GokerPeer) DecryptRoundDeckWithPayload(payload string) {
	pKeys := p.Keyring.GetKeysFromPayload(payload)

	for i := range p.Deck.RoundDeck {
		if !p.gameState.Contains(p.Deck.RoundDeck[i].CardKeys, pKeys[i].String()) {
			p.Keyring.DecryptWithKey(p.Deck.RoundDeck[i].CardValue, pKeys[i])
			p.Deck.RoundDeck[i].CardKeys = append(p.Deck.RoundDeck[i].CardKeys, pKeys[i].String())
		}
	}
}
