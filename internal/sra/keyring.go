package sra

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
)

type Keyring struct {
	// Signature information //
	signingPrivKey *rsa.PrivateKey
	signingPubKey  *rsa.PublicKey
	Otherskeys     map[peer.ID]*rsa.PublicKey

	// SRA infomation //
	sharedP, sharedQ *big.Int

	// Global keys (one set for encrypting every card)
	globalPrivateKey, globalPublicKey, globalN, globalPHI *big.Int

	// Variations of the global keys
	keyVariations []*KeyVariation

	// Time locking
	TLP            *TimeLock
	KeyringPayload string

	BrokenPuzzlePayloads []string // Save all broken puzzles from others for when eval happens
}

// Generate RSA keys for signatures
func (k *Keyring) GenerateSigningKeys() error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate signing keys: %v", err)
	}

	k.signingPrivKey = privateKey
	k.signingPubKey = &privateKey.PublicKey
	return nil
}

// Export the public key as a PEM-encoded string -- will be sent through the network
func (k *Keyring) ExportPublicKey() (string, error) {
	pubASN1, err := x509.MarshalPKIXPublicKey(k.signingPubKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %v", err)
	}

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	})
	return string(pubPEM), nil
}

// DecodePEMPublicKey converts a PEM-encoded string into an *rsa.PublicKey
func DecodePEMPublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("invalid PEM data")
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not an RSA public key")
	}

	return rsaPubKey, nil
}

// Sign a message using RSA private key
func (k *Keyring) SignMessage(message string) (string, error) {
	if k.signingPrivKey == nil {
		return "", fmt.Errorf("signing key not initialized")
	}

	// Hash the message
	hash := sha256.Sum256([]byte(message))

	// Sign the hash
	signature, err := rsa.SignPKCS1v15(rand.Reader, k.signingPrivKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %v", err)
	}

	// Encode signature as base64 for easy transmission
	return base64.StdEncoding.EncodeToString(signature), nil
}

// Verify a message's signature using the sender's public key
func (k *Keyring) VerifySignature(sendingPeer peer.ID, message string, signature string) bool {
	peerPubKey, exists := k.Otherskeys[sendingPeer]
	if !exists {
		log.Fatalf("VerifySignature: missing public key for peer %s\n", sendingPeer)
	}

	// Decode base64 signature
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}

	// Hash the message
	hash := sha256.Sum256([]byte(message))

	// Verify the signature
	err = rsa.VerifyPKCS1v15(peerPubKey, crypto.SHA256, hash[:], sigBytes)
	return err == nil
}

// Function to store a peer's public key in Keyring
func (k *Keyring) SetPeerPublicKey(peerID peer.ID, publicKey string) {
	if k.Otherskeys == nil {
		k.Otherskeys = make(map[peer.ID]*rsa.PublicKey)
	}

	// Decode the PEM public key
	decodedKey, err := DecodePEMPublicKey(publicKey)
	if err != nil {
		log.Printf("Failed to decode public key for peer %s: %v", peerID, err)
		return
	}

	k.Otherskeys[peerID] = decodedKey
	log.Printf("Stored public key for peer: %s", peerID)
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
