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
	CardValue      *big.Int
}

var ranks = [...]string{"ace", "2", "3", "4", "5", "6", "7", "8", "9", "10", "jack", "queen", "king"}
var suits = [...]string{"hearts", "diamonds", "clubs", "spades"}

// GenerateCardHash generates a hash for a card
func generateCardHash(card string, secretKey string) *big.Int {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(card))
	hBytes := h.Sum(nil) // 32 bytes (SHA-256 output)

	// Apply padding (PKCS#1-style, leading zeroes for RSA)
	//paddedHash := make([]byte, 256)
	//copy(paddedHash[256-len(hBytes):], hBytes) // Right-align hash in the padded array

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
	//cardBytes := cardHash.Bytes() // Convert big.Int back to bytes

	// Remove leading zeros (unpadding)
	//unpaddedHash := new(big.Int).SetBytes(cardBytes)
	for k, v := range d.ReferenceDeck {
		// Unpad stored deck hashes for comparison
		//storedBytes := v.Bytes()
		//storedUnpadded := new(big.Int).SetBytes(storedBytes)

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
		if line == "\\END" {
			break
		}
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
		if line == "\\END" {
			break
		}
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
	for _, card := range p.Deck.RoundDeck {
		card.CardValue = p.Keyring.DecryptWithGlobalKeys(card.CardValue)
	}
}

// Encrypt deck with global keys
func (p *GokerPeer) EncryptAllWithGlobalKeys() {
	for _, card := range p.Deck.RoundDeck {
		card.CardValue = p.Keyring.EncryptWithGlobalKeys(card.CardValue)
	}
}

// Encrypt deck with variation numbers
func (p *GokerPeer) EncryptAllWithVariation() {
	log.Println("EncryptAllWithVariation: Function Called")
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

	for _, handInfo := range p.Hands {
		if handInfo.Peer == peerID {
			if len(handInfo.Hand) < 2 {
				log.Println("error: Hand has less than 2 cards for peer")
				continue
			}

			cardOneKey := p.Keyring.GetKeyForCard(handInfo.Hand[0].VariationIndex)
			//log.Println("Peer is requesting key for card: " + handInfo.Hand[0].CardValue.String())
			//log.Println("Key given to peer for that card: " + cardOneKey.String())
			log.Println(handInfo.Hand[0].VariationIndex)
			cardTwoKey := p.Keyring.GetKeyForCard(handInfo.Hand[1].VariationIndex)
			//log.Println("Peer is requesting key for card: " + handInfo.Hand[1].CardValue.String())
			//log.Println("Key given to peer for that card: " + cardTwoKey.String())
			log.Println(handInfo.Hand[1].VariationIndex)

			if cardOneKey == nil || cardTwoKey == nil {
				log.Println("error: Could not retrieve key for one or both cards")
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

	log.Print("KEYS BEING SENT: ")
	fmt.Println(keys)
	return strings.Join(keys, "\n")
}

// Creates the hands array, Should be done once the second step of the protocol is done
func (p *GokerPeer) SetHands() {
	IDs := p.gameState.GetTurnOrder()

	p.Hands = make([]HandInfo, 0, len(IDs))

	for i, id := range IDs {
		cardOne := p.Deck.GetCardFromRoundDeck(i)
		cardTwo := p.Deck.GetCardFromRoundDeck(len(IDs) + i)

		if id == p.ThisHost.ID() { // put this host in another place
			p.MyHand = HandInfo{
				Peer: id,
				Hand: []CardInfo{*cardOne, *cardTwo},
			}
			continue
		}

		p.Hands = append(p.Hands, HandInfo{
			Peer: id,
			Hand: []CardInfo{*cardOne, *cardTwo},
		})
	}

	p.DumpRoundDeck() // To see what the FUCK is going on
}

// Decrypts my hand in the hands array given to key strings
func (p *GokerPeer) DecryptMyHand(cardOneKeys []string, cardTwoKeys []string) {
	log.Println(cardOneKeys)
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

	log.Println(cardTwoKeys)
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
	//fmt.Println("Card 1 after full decryption: " + p.MyHand.Hand[0].CardValue.String())

	plaintextCardTwo, err := p.Keyring.DecryptWithVariation(p.MyHand.Hand[1].CardValue, p.MyHand.Hand[1].VariationIndex)
	if err != nil {
		return
	}
	p.MyHand.Hand[1].CardValue = plaintextCardTwo
	//fmt.Println("Card 2 after full decryption: " + p.MyHand.Hand[0].CardValue.String())
}

func (p *GokerPeer) DumpRoundDeck() {
	if p.Deck == nil || len(p.Deck.RoundDeck) == 0 {
		log.Println("DumpRoundDeck: Round deck is empty or uninitialized")
		return
	}

	log.Println("Dumping Round Deck:")
	for i, card := range p.Deck.RoundDeck {
		log.Printf("Card %d: Value = %s, Variation Index = %d\n", i, card.CardValue.String(), card.VariationIndex)
	}
}
