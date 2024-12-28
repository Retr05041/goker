package game

import (
	"crypto/hmac"
	"crypto/sha256"
	"math/big"
)

type Card struct {
	index          int
	VariationIndex int
	Cardvalue      *big.Int
}

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

