package sra

import (
	"fmt"
	"math/big"
	"testing"
)

// TestGeneratePQ ensures P and Q are generated and are prime.
func TestGeneratePQ(t *testing.T) {
	k := &Keyring{}
	k.GeneratePQ()

	if k.sharedP == nil || k.sharedQ == nil {
		t.Fatal("P and Q were not generated")
	}
	if !k.sharedP.ProbablyPrime(20) || !k.sharedQ.ProbablyPrime(20) {
		t.Fatal("Generated P or Q is not prime")
	}
}

// TestGCD ensures the gcd function correctly calculates the greatest common divisor.
func TestGCD(t *testing.T) {
	a := big.NewInt(48)
	b := big.NewInt(18)
	expected := big.NewInt(6)

	result := gcd(a, b)
	if result.Cmp(expected) != 0 {
		t.Errorf("Expected GCD(%d, %d) = %d, got %d", a, b, expected, result)
	}
}

// TestGenerateKeys ensures key generation works correctly.
func TestGenerateKeys(t *testing.T) {
	k := &Keyring{}
	k.GeneratePQ()

	err := k.GenerateKeys()
	if err != nil {
		t.Fatalf("GenerateKeys failed: %v", err)
	}

	if k.globalPrivateKey == nil || k.globalPublicKey == nil || k.globalN == nil || k.globalPHI == nil {
		t.Fatal("Key generation failed: missing key components")
	}
}

// TestGenerateRandomCoPrime ensures the function generates numbers co-prime to x.
func TestGenerateRandomCoPrime(t *testing.T) {
	x := big.NewInt(1001) // 1001 = 7 * 11 * 13, so it has multiple factors
	coPrime, err := generateRandomCoPrime(x)
	if err != nil {
		t.Fatalf("generateRandomCoPrime failed: %v", err)
	}

	if gcd(x, coPrime).Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Generated number %d is not co-prime with %d", coPrime, x)
	}
}

// TestEncryptionDecryption ensures encryption and decryption using global keys.
func TestEncryptionDecryption(t *testing.T) {
	k := &Keyring{}
	k.GeneratePQ()
	k.GenerateKeys()

	original := big.NewInt(123456789)
	encrypted := k.EncryptWithGlobalKeys(original)
	decrypted := k.DecryptWithGlobalKeys(encrypted)

	if decrypted.Cmp(original) != 0 {
		t.Errorf("Decryption failed: expected %d, got %d", original, decrypted)
	}
}

// TestGetSetPQ ensures that P & Q can be serialized and deserialized properly.
func TestGetSetPQ(t *testing.T) {
	k := &Keyring{}
	k.GeneratePQ()

	pqString := k.GetPQString()

	newK := &Keyring{}
	var p, q string
	fmt.Sscanf(pqString, "%s\n%s\n", &p, &q)
	newK.SetPQ(p, q)

	if k.sharedP.Cmp(newK.sharedP) != 0 || k.sharedQ.Cmp(newK.sharedQ) != 0 {
		t.Errorf("SetPQ failed: expected (%s, %s), got (%s, %s)", k.sharedP, k.sharedQ, newK.sharedP, newK.sharedQ)
	}
}
