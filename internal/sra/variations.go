package sra

import (
	"fmt"
	"math/big"
	"strconv"
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
		fmt.Println("Generating variation: " + strconv.Itoa(i))
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

func TestVariations() {
	k := &Keyring{}
	k.GeneratePQ()
	k.GenerateKeys()

	// Generate 52 variations
	err := k.GenerateKeyVariations(52)
	if err != nil {
		fmt.Println("Error generating key variations:", err)
	}

	// Encrypt with variation 0
	message := HashMessage("Hello, World!")
	cipherText, _ := k.EncryptWithVariation(message, 0)

	// Decrypt with variation 0
	plainText, _ := k.DecryptWithVariation(cipherText, 0)

	// Verify
	fmt.Println("Original:", message)
	fmt.Println("Decrypted:", plainText)
}
