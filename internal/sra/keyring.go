package sra

import (
	"fmt"
	"log"
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
	p, q, err := generateLargePrime(2048)
	if err != nil {
		fmt.Println(err)
		return
	}

	k.sharedP = p
	k.sharedQ = q
}

// Key generation - returns: private key, public key, modulus
// The given p and q are two large primes the players have agreed on - this will create keys that are commutative
// This function also sets the needed 52 Variation keys
func (k *Keyring) GenerateKeys() error {
	if k.sharedP == nil || k.sharedQ == nil {
		return fmt.Errorf("p and q not set")
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

	privateKey := new(big.Int).ModInverse(publicKey, phi)
	if privateKey == nil {
		log.Fatalf("Modular inverse does not exist")
	}

	k.globalPrivateKey, k.globalPublicKey, k.globalN, k.globalPHI = privateKey, publicKey, n, phi
	k.GenerateKeyVariations(52) // We need to create variations each round, so we will do this on Generate Keys
	return nil
}

// Encrypts message (hash) with the global keys inside the keyring.
// Should check if keys exist before attempting
func (k *Keyring) EncryptWithGlobalKeys(data *big.Int) *big.Int {
	return new(big.Int).Exp(data, k.globalPublicKey, k.globalN)
}

// Decyrpts message (hash) with the global keys inside the keyring.
// Should check if keys exist before attempting
func (k *Keyring) DecryptWithGlobalKeys(data *big.Int) *big.Int {
	return new(big.Int).Exp(data, k.globalPrivateKey, k.globalN)
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
