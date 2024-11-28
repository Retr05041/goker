package sra

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
)

// This file is used for locking the payload for time locking

// Encrypts plaintext with AES-256-GCM and returns the ciphertext in base64 format and key
func PayloadToAES(plaintext string) (string, *big.Int, error) {
	key := make([]byte, 32) // AES-256 requires a 32-byte key
	if _, err := rand.Read(key); err != nil {
		fmt.Println("Error generating key:", err)
		return "", nil, nil
	}

	// Generate a new AES cipher 
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create a GCM cipher mode instance
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate a nonce (unique for each encryption operation)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the plaintext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return the ciphertext as a base64-encoded string
	return base64.StdEncoding.EncodeToString(ciphertext), new(big.Int).SetBytes(key), nil
}

// Decrypts a base64-encoded ciphertext using AES-256-GCM
func AESToPayload(ciphertextBase64 string, key *big.Int) (string, error) {
	keyBytes := key.Bytes()

	// Decode the base64-encoded ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 ciphertext: %w", err)
	}

	// Generate a new AES cipher using the provided key
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create a GCM cipher mode instance
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract the nonce from the ciphertext
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt the ciphertext
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt ciphertext: %w", err)
	}

	return string(plaintext), nil
}
