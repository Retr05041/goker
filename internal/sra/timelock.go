package sra

import (
	"fmt"
	"math/big"
	"time"
)

// Time locking functions for private keys - Uses a 2^2^t mod n - Need to change this to be symmetric encrypted...?

// GenerateTimeLockedPrivateKey locks the global private key with a time-lock puzzle.
func (k *Keyring) GenerateTimeLockedPrivateKey(T int64) error {
	// Calculate φ(n) = (p-1)(q-1)
	phi := new(big.Int).Mul(new(big.Int).Sub(k.sharedP, big.NewInt(1)), new(big.Int).Sub(k.sharedQ, big.NewInt(1)))

	// Step 1: Determine `t` based on desired decryption delay (T in seconds)
	// Experimentally chosen squaring speed
	squaringSpeed := k.CalibrateSquaringSpeed() // Adjust based on actual hardware speed
	t := new(big.Int).Mul(big.NewInt(T), big.NewInt(squaringSpeed))

	// Step 2: Calculate `e = 2^t mod φ(n)`
	e := new(big.Int).Exp(big.NewInt(2), t, phi)

	// Initial base of 2
	a := big.NewInt(2)

	// Step 4: Compute `b = a^e mod n`
	b := new(big.Int).Exp(a, e, k.globalN)

	// Step 5: Encrypt the global private key with `b`
	encryptedPrivateKey := new(big.Int).Add(k.globalPrivateKey, b)
	encryptedPrivateKey.Mod(encryptedPrivateKey, k.globalN)

	// Store encrypted key and other values for decryption
	k.globalPrivateKey = encryptedPrivateKey
	k.encryptionIterations = t // Save `t` for decryption

	return nil
}

// DecryptTimeLockedPrivateKey unlocks the time-locked private key by performing
// `t` sequential squaring operations.
func (k *Keyring) DecryptTimeLockedPrivateKey() error {
	if k.globalPrivateKey == nil  || k.encryptionIterations == nil {
		return fmt.Errorf("Encryption parameters must be set before decryption")
	}

	// Step 1: Set base
	b := big.NewInt(2)

	// Step 2: Perform `t` squarings of `b` modulo `n` (sequential squaring)
	for i := big.NewInt(0); i.Cmp(k.encryptionIterations) < 0; i.Add(i, big.NewInt(1)) {
		b.Exp(b, big.NewInt(2), k.globalN)
	}

	// Step 3: Subtract `b` from the encrypted private key to retrieve the original
	k.globalPrivateKey.Sub(k.globalPrivateKey, b)
	k.globalPrivateKey.Mod(k.globalPrivateKey, k.globalN)

	return nil
}

func (k *Keyring) CalibrateSquaringSpeed() int64 { // TODO: Call this ONLY if your calibration is nil
	// Set a large number of squarings to get an accurate measure.
	const numSquarings = 1_000_000 // Adjust as needed for precision and timing.

	// Use a random large integer as the base to square.
	base := big.NewInt(2) // Use a larger, more realistic base.
	
	// Start measuring the time.
	start := time.Now()

	// Perform the squaring operations.
	temp := new(big.Int).Set(base)
	for i := int64(0); i < numSquarings; i++ {
		temp.Exp(temp, big.NewInt(2), k.globalN) // Squaring operation.
	}

	// Measure the elapsed time.
	elapsed := time.Since(start).Seconds()

	// Calculate the squaring speed (operations per second).
	squaringSpeed := int64(float64(numSquarings) / elapsed)

	fmt.Printf("Calibration complete: Estimated squaring speed is %d operations per second\n", squaringSpeed)
	return squaringSpeed
}

// Test function to validate the functionality
func TestTimeLockPuzzle() {
	// Set the desired delay time (in seconds) for the time-lock puzzle
	const T = 60*10 // Adjust T to test different delays

	// Initialize Keyring and generate p and q
	keyring := &Keyring{}
	keyring.GeneratePQ()

	// Generate global keys (public and private)
	keyring.GenerateGlobalKeys()

	// Save the original private key for later comparison
	originalPrivateKey := new(big.Int).Set(keyring.globalPrivateKey)

	// Encrypt the private key with the time-lock puzzle
	fmt.Printf("Encrypting the private key with a time-lock puzzle for %d seconds...\n", T)
	err := keyring.GenerateTimeLockedPrivateKey(T)
	if err != nil {
		fmt.Printf("Failed to generate time-locked private key: %v\n", err)
	}

	// Measure the decryption time
	start := time.Now()

	fmt.Printf("Starting decryption of the time-locked private key...\n")
	err = keyring.DecryptTimeLockedPrivateKey()
	if err != nil {
		fmt.Printf("Failed to decrypt time-locked private key: %v\n", err)
	}

	elapsed := time.Since(start).Seconds()
	fmt.Printf("Decryption completed in approximately %.2f seconds\n", elapsed)

	// Check if the decryption time is close to T (within a margin)
	const timeMargin = 0.1 // Allow a small margin for timing variations
	if elapsed < float64(T)-timeMargin || elapsed > float64(T)+timeMargin {
		fmt.Printf("Decryption time was %.2f seconds, expected around %d seconds\n", elapsed, T)
	}

	// Validate that the decrypted private key matches the original private key
	if keyring.globalPrivateKey.Cmp(originalPrivateKey) != 0 {
		fmt.Println("Decrypted private key does not match the original private key")
	} else {
		fmt.Println("Decrypted private key matches the original private key!")
	}
}
