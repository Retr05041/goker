package sra

// SRA - RSA variant

import (
	"goker/internal/game"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
)

type Keyring struct {
	sharedP, sharedQ *big.Int

	// Global keys (one set for encrypting every card)
	globalPrivateKey, globalPublicKey, globalN, globalPHI *big.Int

	// Variations of the global keys
	keyVariations []*KeyVariation

	// Time locking
	puzzle         *TimeLock
	KeyringPayload string
}

// Set p & q to a randomly generated 2048 bit prime number.
// Used for when a user hosts the game.
func (k *Keyring) GeneratePQ() {
	p, err := generateLargePrime(2048)
	if err != nil {
		fmt.Println(err)
	}
	q, err := generateLargePrime(2048)
	if err != nil {
		fmt.Println(err)
	}

	k.sharedP = p
	k.sharedQ = q
}

// Function to generate two large prime numbers
func generateLargePrime(bits int) (*big.Int, error) {
	return rand.Prime(rand.Reader, bits)
}

// Greatest Common Divisor between 2 numbers and return it
func gcd(a, b *big.Int) *big.Int {
	return new(big.Int).GCD(nil, nil, a, b)
}

// Modular inverse of a modulo m - using Extended Euclidean Algorithm
// return x, such that `(a*x) mod m = 1`
// given a public key and Eulers Totient, it will return a private key
func modInverse(a, m *big.Int) (*big.Int, error) {
	mCopy := new(big.Int).Set(m) // store m for later use
	// These will represent the coefficients of the linear combination
	y := big.NewInt(0)
	x := big.NewInt(1)

	// every number is congruent to 0 modulo 1
	if mCopy.Cmp(big.NewInt(1)) == 0 {
		return nil, fmt.Errorf("modular inverse does not exist")
	}

	for a.Cmp(big.NewInt(1)) > 0 {
		quotent := new(big.Int).Div(a, mCopy)
		t := new(big.Int).Set(mCopy) // store m for this iteration

		mCopy.Set(new(big.Int).Mod(a, mCopy)) // m = a mod m
		a = t                                 // set previous value of m becomes a
		t = new(big.Int).Set(y)               // set the hold var to old y

		y = new(big.Int).Set(new(big.Int).Sub(x, new(big.Int).Mul(quotent, y))) // y = x-(quotent * m)
		x = t                                                                   // x now becomes old y
	}

	// if x is negative, adjust by adding m0 to ensure the result is positive
	if x.Cmp(big.NewInt(0)) < 0 {
		x = new(big.Int).Add(x, m)
	}

	return x, nil
}

// Key generation - returns: private key, public key, modulus
// The given p and q are two large primes the players have agreed on - this will create keys that are commutative
func (k *Keyring) GenerateKeys() error {
	if k.sharedP == nil || k.sharedQ == nil {
		return fmt.Errorf("P and Q not set.")
	}
	// n = p * q
	n := new(big.Int).Mul(k.sharedP, k.sharedQ)

	// Eulers Totient -- ϕ(n) = (p−1)(q−1)
	// Used for caluclating private keys
	phi := new(big.Int).Mul(new(big.Int).Sub(k.sharedP, big.NewInt(1)), new(big.Int).Sub(k.sharedQ, big.NewInt(1)))

	var publicKey *big.Int

	publicKey, err := generateRandomCoPrime(phi)
	if err != nil {
		return err
	}

	privateKey, err := modInverse(publicKey, phi)
	if err != nil {
		return err
	}

	k.globalPrivateKey, k.globalPublicKey, k.globalN, k.globalPHI = privateKey, publicKey, n, phi
	return nil
}

// Generates a random number and checks if it is co-prime with x
// Used when generating keys, x will always be Eulers Totient
func generateRandomCoPrime(x *big.Int) (*big.Int, error) {
	for {
		// Generate a random number in the range [2, max)
		randomNum, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), uint(2048))) // Limit size is 2048 bits
		if err != nil {
			return nil, err
		}
		randomNum.Add(randomNum, big.NewInt(2)) // Adjust to ensure it's at least 2

		// Check if GCD is 1
		if gcd(x, randomNum).Cmp(big.NewInt(1)) == 0 {
			return randomNum, nil
		}
	}
}

func HashMessage(message string) *big.Int {
	hash := sha256.Sum256([]byte(message))
	hashBigInt := new(big.Int).SetBytes(hash[:])
	return hashBigInt
}

// Encrypts message (hash) with the global keys inside the keyring.
// Should check if keys exist before attempting
func (k *Keyring) EncryptWithGlobalKeys(data *big.Int) *big.Int {
	return new(big.Int).Exp(data, k.globalPublicKey, k.globalN)
}

// Same as EncryptWithGlobalKeys but over a list of Cards
func (k *Keyring) EncryptAllWithGlobalKeys(data []game.Card) []game.Card {
	for i, v := range data {
		data[i].Cardvalue = k.EncryptWithGlobalKeys(v.Cardvalue)
	}
	return data
}

// Decyrpts message (hash) with the global keys inside the keyring.
// Should check if keys exist before attempting
func (k *Keyring) DecryptWithGlobalKeys(data *big.Int) *big.Int {
	return new(big.Int).Exp(data, k.globalPrivateKey, k.globalN)
}

// Use on hash after full decryption to pad it back to full and return it as a string
func PadHash(hash *big.Int) string {
	// Convert back to bytes
	decryptedHash := hash.Bytes()

	// Pad the decrypted hash to match the original hash size
	expectedHash := make([]byte, sha256.Size)
	copy(expectedHash[sha256.Size-len(decryptedHash):], decryptedHash)

	return string(expectedHash)
}

// Returns P&Q as a string - meant for being sent over a stream for a PQRequest
func (k *Keyring) GetPQString() string {
	return fmt.Sprintf("%s\n%s\n", k.sharedP, k.sharedQ)
}

// Set p and q
func (k *Keyring) SetPQ(p string, q string) {
	k.sharedP = new(big.Int)
	k.sharedQ = new(big.Int)
	k.sharedP.SetString(p, 10)
	k.sharedQ.SetString(q, 10)
}
