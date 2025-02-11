package sra

import (
	"fmt"
	"math/big"
)

type KeyVariation struct {
	publicKey, privateKey, variationValue *big.Int
}

func (k *Keyring) GenerateKeyVariations(count int) error {
	if k.globalPrivateKey == nil || k.globalPublicKey == nil || k.globalN == nil {
		return fmt.Errorf("global keys not generated")
	}

	k.keyVariations = make([]*KeyVariation, count)

	for i := 0; i < count; i++ {
		currentVariation := new(KeyVariation)

		r, err := generateRandomCoPrime(k.globalPHI)
		if err != nil {
			return err
		}

		// Use a copy of phi when calling modInverse
		rInv, err := modInverse(r, k.globalPHI)
		if err != nil {
			return err
		}

		currentVariation.publicKey = new(big.Int).Mod(new(big.Int).Mul(k.globalPublicKey, r), k.globalPHI)
		currentVariation.privateKey = new(big.Int).Mod(new(big.Int).Mul(k.globalPrivateKey, rInv), k.globalPHI)
		currentVariation.variationValue = r
		k.keyVariations[i] = currentVariation
	}

	return nil
}

func (k *Keyring) EncryptWithVariation(data *big.Int, index int) (*big.Int, error) {
	if index >= len(k.keyVariations) {
		return nil, fmt.Errorf("invalid key variation index")
	}
	return new(big.Int).Exp(data, k.keyVariations[index].publicKey, k.globalN), nil
}

func (k *Keyring) DecryptWithVariation(data *big.Int, index int) (*big.Int, error) {
	if index >= len(k.keyVariations) {
		return nil, fmt.Errorf("invalid key variation index")
	}
	return new(big.Int).Exp(data, k.keyVariations[index].privateKey, k.globalN), nil
}

func (k *Keyring) GetKeyForCard(variationIndex int) *big.Int {
	return k.keyVariations[variationIndex].privateKey
}

func (k *Keyring) GenerateKeyringPayload() error {
	if k.globalPublicKey == nil || k.globalPrivateKey == nil || k.keyVariations == nil {
		return fmt.Errorf("error: Missing keys or variations")
	}

	// Start with global public and private keys
	payload := fmt.Sprintf("%s\n%s\n", k.globalPublicKey.String(), k.globalPrivateKey.String())

	// Append each variation's `r` value
	for _, variation := range k.keyVariations {
		if variation != nil {
			payload += fmt.Sprintf("%s\n", variation.variationValue.String())
		}
	}

	k.KeyringPayload = payload
	return nil
}
