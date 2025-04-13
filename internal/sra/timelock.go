package sra

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"time"
)

// Structure for Time locked puzzle payload
type TimeLock struct {
	Puzzle  string `json:"puzzle"`
	Payload string `json:"payload"`
	Iter    string `json:"iterations"`
	N       string `json:"n"`
}

// Time locking functions for private keys - Uses a 2^2^t mod n - Need to change this to be symmetric encrypted...?

// GenerateTimeLockedPrivateKey locks a big int for a specified amount of time
// In this context, it should be used on the symmetric key that locks all xn's (which hold the details for the variations of the global keyring)
func (k *Keyring) GenerateTimeLockedPuzzle(seconds int64) {
	k.TLP = new(TimeLock)

	// Encrypt Payload with AES to lock key
	encryptedPayload, key, err := PayloadToAES(k.KeyringPayload)
	if err != nil {
		fmt.Println(err)
	}

	// P&Q - will be burned
	p, q, err := generateLargePrime(2048)
	if err != nil {
		fmt.Println(err)
	}

	// mod n - THIS WILL NEED TO BE KEPT
	n := new(big.Int).Mul(p, q)

	// Calculate φ(n) = (p-1)(q-1) - THIS WILL NEED TO BE KEPT
	phi := new(big.Int).Mul(new(big.Int).Sub(p, big.NewInt(1)), new(big.Int).Sub(q, big.NewInt(1)))

	// Step 1: Determine `t` based on desired decryption delay (T in seconds)
	// Experimentally chosen squaring speed
	squaringSpeed := k.squaringSpeed // Adjust based on actual hardware speed
	iterations := new(big.Int).Mul(big.NewInt(seconds), big.NewInt(squaringSpeed))

	// Step 2: Calculate `e = 2^t mod φ(n)`
	e := new(big.Int).Exp(big.NewInt(2), iterations, phi)

	// Initial base of 2
	a := big.NewInt(2)

	// Step 4: Compute `b = a^e mod n`
	b := new(big.Int).Exp(a, e, n)

	// Step 5: Encrypt the global private key with `b`
	lockedPuzzle := new(big.Int).Add(key, b)
	lockedPuzzle.Mod(lockedPuzzle, n)

	// Return made puzzle
	k.TLP.Puzzle = lockedPuzzle.String()
	k.TLP.Payload = encryptedPayload
	k.TLP.Iter = iterations.String()
	k.TLP.N = n.String()
}

func (k *Keyring) CalibrateSquaringSpeed() {
	p, q, err := generateLargePrime(2048)
	if err != nil {
		fmt.Println(err)
	}
	n := new(big.Int).Mul(p, q)

	// Set a large number of squarings to get an accurate measure.
	const numSquarings = 1_000_000 // Adjust as needed for precision and timing.

	// Use a random large integer as the base to square.
	base := big.NewInt(2) // Use a larger, more realistic base.

	// Start measuring the time.
	start := time.Now()

	// Perform the squaring operations.
	temp := new(big.Int).Set(base)
	for i := int64(0); i < numSquarings; i++ {
		temp.Exp(temp, big.NewInt(2), n) // Squaring operation.
	}

	// Measure the elapsed time.
	elapsed := time.Since(start).Seconds()

	// Calculate the squaring speed (operations per second).
	squaringSpeed := int64(float64(numSquarings) / elapsed)

	fmt.Printf("Calibration complete: Estimated squaring speed is %d operations per second\n", squaringSpeed)
	k.squaringSpeed = squaringSpeed
}

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
