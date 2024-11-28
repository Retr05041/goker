package sra

import (
	"fmt"
	"math/big"
	"time"
)


type TimeLock struct {
	lockedPuzzle *big.Int // Shared puzzle of key
	payload string  // Base64 encrypted payload string
	
	encryptionIterations *big.Int // Shared
	n *big.Int // Shared

	phi *big.Int // SHARED ONLY FOR SPEED OF BREAKING
}

// Time locking functions for private keys - Uses a 2^2^t mod n - Need to change this to be symmetric encrypted...?

// GenerateTimeLockedPrivateKey locks a big int for a specified amount of time
// In this context, it should be used on the symmetric key that locks all xn's (which hold the details for the variations of the global keyring)
func (k *Keyring) GenerateTimeLockedPuzzle(payload string, seconds int64) *TimeLock {
	timelock := new(TimeLock)
	
	// Encrypt Payload with AES to lock key
	encryptedPayload, key, err := PayloadToAES(payload)
	if err != nil {
		fmt.Println(err)
	}

	// P&Q - will be burned
	p, err := generateLargePrime(2048)
	if err != nil {
		fmt.Println(err)
	}
	q, err := generateLargePrime(2048)
	if err != nil {
		fmt.Println(err)
	}

	// mod n - THIS WILL NEED TO BE KEPT
	timelock.n = new(big.Int).Mul(p, q) 

	// Calculate φ(n) = (p-1)(q-1) - THIS WILL NEED TO BE KEPT
	timelock.phi = new(big.Int).Mul(new(big.Int).Sub(p, big.NewInt(1)), new(big.Int).Sub(q, big.NewInt(1)))

	// Step 1: Determine `t` based on desired decryption delay (T in seconds)
	// Experimentally chosen squaring speed
	squaringSpeed := k.CalibrateSquaringSpeed(timelock.n) // Adjust based on actual hardware speed
	timelock.encryptionIterations = new(big.Int).Mul(big.NewInt(seconds), big.NewInt(squaringSpeed))

	// Step 2: Calculate `e = 2^t mod φ(n)`
	e := new(big.Int).Exp(big.NewInt(2), timelock.encryptionIterations, timelock.phi)

	// Initial base of 2
	a := big.NewInt(2)

	// Step 4: Compute `b = a^e mod n`
	b := new(big.Int).Exp(a, e, timelock.n)

	// Step 5: Encrypt the global private key with `b`
	timelock.lockedPuzzle = new(big.Int).Add(key, b)
	timelock.lockedPuzzle.Mod(timelock.lockedPuzzle, timelock.n)


	timelock.payload = encryptedPayload
	return timelock
}

// Unlocks the time-locked key by performing
// `t` sequential squaring operations.
func (k *Keyring) BreakTimeLockedPuzzle(puzzle *big.Int, encryptedPayload string, decryptionIterations *big.Int, n *big.Int) string {
	// Step 1: Set base
	base := big.NewInt(2)

	// Step 2: Perform 't' squarings of 'base' modulo 'n' 
	for i := big.NewInt(0); i.Cmp(decryptionIterations) < 0; i.Add(i, big.NewInt(1)) {
		base.Exp(base, big.NewInt(2), n)
	}

	// Step 3: Subtract `b` from the time locked puzzle to retrieve the private key
	key := new(big.Int).Sub(puzzle, base)
	key.Mod(key, n) 


	plaintextPayload, err := AESToPayload(encryptedPayload, key)
	if err != nil {
		fmt.Println(err)
	}
	return plaintextPayload
}

func (k *Keyring) CalibrateSquaringSpeed(n *big.Int) int64 { 
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
	return squaringSpeed
}

// Test function to validate the functionality
func TestTimeLockPuzzle() {
	// Set the desired delay time (in seconds) for the time-lock puzzle
	const T = 30 // Adjust T to test different delays

	// Initialize Keyring and generate p and q
	keyring := &Keyring{}
	keyring.GeneratePQ()

	// Generate global keys (public and private)
	keyring.GenerateKeys()

	// Create variations (assuming we need them to encrypt the cards)
	keyring.GenerateKeyVariations(5)
	
	// Create payload
	keyring.GenerateKeyringPayload()
	fmt.Println(keyring.KeyringPayload)

	// Lock payload with a time lock
	fmt.Printf("Encrypting the payload with a time-lock puzzle for %d seconds...\n", T)
	keyring.puzzle = keyring.GenerateTimeLockedPuzzle(keyring.KeyringPayload, T)


	// Measure the decryption time
	start := time.Now()

	fmt.Printf("Starting decryption of the time-locked payload...\n")
	plainTextPayload := keyring.BreakTimeLockedPuzzle(keyring.puzzle.lockedPuzzle, keyring.puzzle.payload, keyring.puzzle.encryptionIterations, keyring.puzzle.n)

	elapsed := time.Since(start).Seconds()
	fmt.Printf("Decryption completed in approximately %.2f seconds\n", elapsed)

	// Check if the decryption time is close to T (within a margin)
	const timeMargin = 0.1 // Allow a small margin for timing variations
	if elapsed < float64(T)-timeMargin || elapsed > float64(T)+timeMargin {
		fmt.Printf("Decryption time was %.2f seconds, expected around %d seconds\n", elapsed, T)
	}

	// Validate that the decrypted private key matches the original private key
	if plainTextPayload != keyring.KeyringPayload {
		fmt.Println("Decrypted payload does not match the original payload")
	} else {
		fmt.Println("Decrypted payload matches the original payload!")
	}
}
