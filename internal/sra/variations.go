package sra

import (
	"fmt"
	"log"
	"math/big"
	"strings"
)

type KeyVariation struct {
	publicKey, privateKey, variationValue *big.Int
}

func (k *Keyring) GenerateKeyVariations(count int) error {
	if k.globalPrivateKey == nil || k.globalPublicKey == nil || k.globalN == nil {
		return fmt.Errorf("global keys not generated")
	}

	k.keyVariations = make([]*KeyVariation, count)

	for i := range count {
		r, err := generateRandomCoPrime(k.globalPHI)
		if err != nil {
			return err
		}

		rInv := new(big.Int).ModInverse(r, k.globalPHI)
		if rInv == nil {
			log.Fatalf("Modular inverse does not exist for variation %d", i)
		}

		k.keyVariations[i] = &KeyVariation{
			publicKey:      new(big.Int).Mod(new(big.Int).Mul(k.globalPublicKey, r), k.globalPHI),
			privateKey:     new(big.Int).Mod(new(big.Int).Mul(k.globalPrivateKey, rInv), k.globalPHI),
			variationValue: r,
		}
	}

	return nil
}

func (k *Keyring) EncryptWithVariation(data *big.Int, index int) error {
	if index < 0 || index >= len(k.keyVariations) || k.keyVariations[index] == nil {
		return fmt.Errorf("invalid key variation index")
	}

	data.Exp(data, k.keyVariations[index].publicKey, k.globalN)
	return nil
}

func (k *Keyring) DecryptWithVariation(data *big.Int, index int) error {
	if index >= len(k.keyVariations) {
		return fmt.Errorf("invalid key variation index")
	}

	data.Exp(data, k.keyVariations[index].privateKey, k.globalN)
	return nil
}

func (k *Keyring) DecryptWithKey(data *big.Int, key *big.Int) {
	data.Exp(data, key, k.globalN)
}

func (k *Keyring) GetVariationKeyForCard(variationIndex int) *big.Int {
	return k.keyVariations[variationIndex].privateKey
}

func (k *Keyring) GenerateKeyringPayload() error {
	var payload []string
	if k.globalPublicKey == nil || k.globalPrivateKey == nil || k.keyVariations == nil {
		return fmt.Errorf("error: Missing keys or variations")
	}

	// Start with global public and private keys
	payload = append(payload, k.globalPublicKey.String(), k.globalPrivateKey.String())

	// Append each variation's `r` value
	for _, variation := range k.keyVariations {
		if variation != nil {
			payload = append(payload, variation.variationValue.String())
		}
	}

	k.KeyringPayload = strings.Join(payload, "\n")
	return nil
}
