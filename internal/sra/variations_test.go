package sra

import (
	"math/big"
	"testing"
)

// TestGenerateKeyVariations ensures key variations are correctly generated.
func TestGenerateKeyVariations(t *testing.T) {
	k := &Keyring{}
	k.GeneratePQ()
	k.GenerateKeys()

	err := k.GenerateKeyVariations(5)
	if err != nil {
		t.Fatalf("GenerateKeyVariations failed: %v", err)
	}

	if len(k.keyVariations) != 5 {
		t.Errorf("Expected 5 key variations, got %d", len(k.keyVariations))
	}

	for _, variation := range k.keyVariations {
		if variation == nil || variation.publicKey == nil || variation.privateKey == nil || variation.variationValue == nil {
			t.Error("Key variation contains nil values")
		}
	}
}

// TestEncryptWithVariation ensures encryption works with key variations.
func TestEncryptWithVariation(t *testing.T) {
	k := &Keyring{}
	k.GeneratePQ()
	k.GenerateKeys()
	k.GenerateKeyVariations(5)

	data := big.NewInt(12345)
	encrypted, err := k.EncryptWithVariation(data, 2)
	if err != nil {
		t.Fatalf("EncryptWithVariation failed: %v", err)
	}

	if encrypted.Cmp(data) == 0 {
		t.Error("Encryption failed: Encrypted value matches original data")
	}
}

// TestDecryptWithVariation ensures decryption with variations works correctly.
func TestDecryptWithVariation(t *testing.T) {
	k := &Keyring{}
	k.GeneratePQ()
	k.GenerateKeys()
	k.GenerateKeyVariations(5)

	data := big.NewInt(12345)
	encrypted, _ := k.EncryptWithVariation(data, 2)
	decrypted, err := k.DecryptWithVariation(encrypted, 2)
	if err != nil {
		t.Fatalf("DecryptWithVariation failed: %v", err)
	}

	if decrypted.Cmp(data) != 0 {
		t.Errorf("Decryption failed: expected %d, got %d", data, decrypted)
	}
}

// TestDecryptWithKey ensures manual decryption works with a given key.
func TestDecryptWithKey(t *testing.T) {
	k := &Keyring{}
	k.GeneratePQ()
	k.GenerateKeys()

	data := big.NewInt(67890)
	encrypted := k.EncryptWithGlobalKeys(data)
	decrypted := k.DecryptWithKey(encrypted, k.globalPrivateKey)

	if decrypted.Cmp(data) != 0 {
		t.Errorf("DecryptWithKey failed: expected %d, got %d", data, decrypted)
	}
}

// TestGetKeyForCard ensures the correct key is returned for a given variation.
func TestGetKeyForCard(t *testing.T) {
	k := &Keyring{}
	k.GeneratePQ()
	k.GenerateKeys()
	k.GenerateKeyVariations(5)

	key := k.GetKeyForCard(3)
	if key == nil {
		t.Error("GetKeyForCard returned nil")
	}

	if key.Cmp(k.keyVariations[3].privateKey) != 0 {
		t.Errorf("GetKeyForCard returned incorrect key")
	}
}
