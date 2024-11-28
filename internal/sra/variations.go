package sra

import (
	"fmt"
	"math/big"
	"strconv"
)

func (k *Keyring) GenerateKeyVariations(count int) error {
    if k.globalPrivateKey == nil || k.globalPublicKey == nil || k.globalN == nil {
        return fmt.Errorf("global keys not generated")
    }

    k.publicKeyVariations = make([]*big.Int, count)
    k.privateKeyVariations = make([]*big.Int, count)

    for i := 0; i < count; i++ {
		fmt.Println("Generating variation: " + strconv.Itoa(i))
        r, err := generateRandomCoPrime(k.globalPHI)
        if err != nil  {
            return err
        }

        // Use a copy of phi when calling modInverse
        rInv, err := modInverse(r, k.globalPHI)
        if err != nil {
            return err
        }

        k.publicKeyVariations[i] = new(big.Int).Mod(new(big.Int).Mul(k.globalPublicKey, r), k.globalPHI)
        k.privateKeyVariations[i] = new(big.Int).Mod(new(big.Int).Mul(k.globalPrivateKey, rInv), k.globalPHI)
    }

    return nil
}


func (k *Keyring) EncryptWithVariation(data *big.Int, index int) (*big.Int, error) {
    if index >= len(k.publicKeyVariations) {
        return nil, fmt.Errorf("invalid key variation index")
    }
    return new(big.Int).Exp(data, k.publicKeyVariations[index], k.globalN), nil
}

func (k *Keyring) DecryptWithVariation(data *big.Int, index int) (*big.Int, error) {
    if index >= len(k.privateKeyVariations) {
        return nil, fmt.Errorf("invalid key variation index")
    }
    return new(big.Int).Exp(data, k.privateKeyVariations[index], k.globalN), nil
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
