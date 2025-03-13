package sra

import (
	"fmt"
	"log"
	"math/big"
	"strings"
)

type Keyring struct {
	sharedP, sharedQ *big.Int

	// Global keys (one set for encrypting every card)
	globalPrivateKey, globalPublicKey, globalN, globalPHI *big.Int

	// Variations of the global keys
	keyVariations []*KeyVariation

	// Time locking
	TLP            *TimeLock
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
	k.GenerateKeyringPayload()  // Get time locked puzzle setup
	return nil
}

// Encrypts data with the global keys inside the keyring.
func (k *Keyring) EncryptWithGlobalKeys(data *big.Int) {
	data.Exp(data, k.globalPublicKey, k.globalN)
}

// Decyrpts data with the global keys inside the keyring.
func (k *Keyring) DecryptWithGlobalKeys(data *big.Int) {
	data.Exp(data, k.globalPrivateKey, k.globalN)
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

// Given a payload for someones entire keyring, give me the actual private keys (for decryption)
func (k *Keyring) GetKeysFromPayload(payload string) []*big.Int {
	keyList := strings.Split(payload, "\n")
	var keys []*big.Int
	privKey, ok := new(big.Int).SetString(keyList[1], 10)
	if !ok {
		log.Fatalf("Couldn't convert private key from payload")
	}

	for i, key := range keyList {
		if i == 0 || i == 1 { // Don't care about the global keys right now
			continue
		}

		r, ok := new(big.Int).SetString(key, 10)
		if !ok {
			log.Fatalf("Cannot set variation number")
		}

		rInv := new(big.Int).ModInverse(r, k.globalPHI)
		keys = append(keys, new(big.Int).Mod(new(big.Int).Mul(privKey, rInv), k.globalPHI))
	}

	return keys
}
