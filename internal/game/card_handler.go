package game

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateCardHash generates a hash for a card
func GenerateCardHash(card string, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(card))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyCardHash verifies if the hash matches the card
func VerifyCardHash(card string, secretKey string, hash string) bool {
	expectedHash := GenerateCardHash(card, secretKey)
	return hmac.Equal([]byte(expectedHash), []byte(hash))
}

func testGame() {
	secretKey := "mySecretKey"
	card := "Ace of Spades"

	// Generate hash for the card
	hash := GenerateCardHash(card, secretKey)
	fmt.Printf("Card: %s, Hash: %s\n", card, hash)

	// Verify the card hash
	isValid := VerifyCardHash(card, secretKey, hash)
	fmt.Printf("Is the hash valid? %v\n", isValid)
}
