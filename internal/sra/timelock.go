package sra

import (
	"fmt"
	"math/big"
	"time"
)


type TimeLock struct {
	lockedPuzzle *big.Int

	encryptionIterations *big.Int
	n *big.Int
	phi *big.Int
}

// Time locking functions for private keys - Uses a 2^2^t mod n - Need to change this to be symmetric encrypted...?

// GenerateTimeLockedPrivateKey locks the global private key with a time-lock puzzle.
func (k *Keyring) GenerateTimeLockedPuzzle(payload *big.Int, seconds int64) *TimeLock {
	timelock := new(TimeLock)

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
	timelock.lockedPuzzle = new(big.Int).Add(payload, b)
	timelock.lockedPuzzle.Mod(timelock.lockedPuzzle, timelock.n)


	return timelock
}

// DecryptTimeLockedPrivateKey unlocks the time-locked private key by performing
// `t` sequential squaring operations.
func (k *Keyring) BreakTimeLockedPuzzle(payload *big.Int, decryptionIterations *big.Int, n *big.Int) *big.Int {
	// Step 1: Set base
	base := big.NewInt(2)

	// Step 2: Perform 't' squarings of 'base' modulo 'n' 
	for i := big.NewInt(0); i.Cmp(decryptionIterations) < 0; i.Add(i, big.NewInt(1)) {
		base.Exp(base, big.NewInt(2), n)
	}

	// Step 3: Subtract `b` from the time locked puzzle to retrieve the private key
	answer := new(big.Int).Sub(payload, base)
	answer.Mod(answer, n) 

	return answer
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
	keyring.GenerateGlobalKeys()

	// Encrypt the private key with the time-lock puzzle
	fmt.Printf("Encrypting the private key with a time-lock puzzle for %d seconds...\n", T)
	keyring.puzzle = keyring.GenerateTimeLockedPuzzle(keyring.globalPrivateKey, T)


	// Measure the decryption time
	start := time.Now()

	fmt.Printf("Starting decryption of the time-locked private key...\n")
	answer := keyring.BreakTimeLockedPuzzle(keyring.puzzle.lockedPuzzle, keyring.puzzle.encryptionIterations, keyring.puzzle.n)

	elapsed := time.Since(start).Seconds()
	fmt.Printf("Decryption completed in approximately %.2f seconds\n", elapsed)

	// Check if the decryption time is close to T (within a margin)
	const timeMargin = 0.1 // Allow a small margin for timing variations
	if elapsed < float64(T)-timeMargin || elapsed > float64(T)+timeMargin {
		fmt.Printf("Decryption time was %.2f seconds, expected around %d seconds\n", elapsed, T)
	}

	// Validate that the decrypted private key matches the original private key
	if answer.Cmp(keyring.globalPrivateKey) != 0 {
		fmt.Println("Decrypted private key does not match the original private key")
	} else {
		fmt.Println("Decrypted private key matches the original private key!")
	}
}
