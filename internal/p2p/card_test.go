package p2p

import (
	"goker/internal/sra"
	"math/big"
	"strings"
	"testing"
)

func TestVerifyCardHash(t *testing.T) {
	card := "hearts_ace"
	key := "testkey"
	hash := generateCardHash(card, key)
	if !VerifyCardHash(card, key, hash) {
		t.Errorf("VerifyCardHash failed for correct hash")
	}

	wrongKey := "wrongkey"
	if VerifyCardHash(card, wrongKey, hash) {
		t.Errorf("VerifyCardHash should fail for incorrect key")
	}
}

func TestVerifyCardHashWithEncryption(t *testing.T) {
	// Step 1: Setup Keyring & Generate Keys
	keyring := &sra.Keyring{}
	keyring.GeneratePQ()
	err := keyring.GenerateKeys()
	if err != nil {
		t.Fatalf("Failed to generate keys: %v", err)
	}

	// Step 2: Generate key variations (for each player)
	err = keyring.GenerateKeyVariations(52)
	if err != nil {
		t.Fatalf("Failed to generate key variations: %v", err)
	}

	// Step 3: Generate card hash
	card := "hearts_ace"
	key := "testkey"
	hash := generateCardHash(card, key)

	// Verify the hash before encryption
	if !VerifyCardHash(card, key, hash) {
		t.Errorf("VerifyCardHash failed for correct hash before encryption")
	}

	// Step 4: Encrypt with Global Keys
	encryptedHash := keyring.EncryptWithGlobalKeys(hash)

	// Step 5: Decrypt with Global Keys
	decryptedHash := keyring.DecryptWithGlobalKeys(encryptedHash)
	if decryptedHash.Cmp(hash) != 0 {
		t.Errorf("Decryption with global keys failed, expected %s but got %s", hash.String(), decryptedHash.String())
	}

	// Step 6: Encrypt with a Key Variation
	variationIndex := 0
	encryptedVariation, err := keyring.EncryptWithVariation(hash, variationIndex)
	if err != nil {
		t.Fatalf("Encryption with variation failed: %v", err)
	}

	// Step 7: Decrypt with the same Key Variation
	decryptedVariation, err := keyring.DecryptWithVariation(encryptedVariation, variationIndex)
	if err != nil {
		t.Fatalf("Decryption with variation failed: %v", err)
	}

	// Verify that the decrypted value matches the original hash
	if decryptedVariation.Cmp(hash) != 0 {
		t.Errorf("Variation decryption failed, expected %s but got %s", hash.String(), decryptedVariation.String())
	}

	// Step 8: Ensure VerifyCardHash still works after full encryption cycle
	if !VerifyCardHash(card, key, decryptedVariation) {
		t.Errorf("VerifyCardHash failed after full encryption and decryption cycle")
	}
}

func TestShuffleRoundDeck(t *testing.T) {
	deck := &deckInfo{}
	deck.GenerateRoundDeck("testkey")
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
	deck.GenerateRoundDeck("testkey")
	payload := deck.GenerateDeckPayload()

	if len(strings.Split(payload, "\n")) != 52 {
		t.Errorf("Expected payload with 52 lines, got %d", len(strings.Split(payload, "\n")))
	}
}
