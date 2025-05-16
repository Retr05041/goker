package sra

import (
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAESEncryption(t *testing.T) {
	t.Run("successful round trip", func(t *testing.T) {
		plaintext := "secret message"
		ciphertext, key, err := PayloadToAES(plaintext)
		require.NoError(t, err)
		require.NotEmpty(t, ciphertext)
		require.NotNil(t, key)

		result, err := AESToPayload(ciphertext, key)
		require.NoError(t, err)
		require.Equal(t, plaintext, result)
	})

	t.Run("empty plaintext", func(t *testing.T) {
		ciphertext, key, err := PayloadToAES("")
		require.NoError(t, err)

		result, err := AESToPayload(ciphertext, key)
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("invalid ciphertext", func(t *testing.T) {
		_, err := AESToPayload("invalid base64", big.NewInt(0))
		require.Error(t, err)
	})

	t.Run("wrong key", func(t *testing.T) {
		plaintext := "test message"
		ciphertext, _, _ := PayloadToAES(plaintext)

		// Generate different key
		wrongKey := big.NewInt(12345)
		_, err := AESToPayload(ciphertext, wrongKey)
		require.Error(t, err)
	})
}

func TestTimeLockPuzzle(t *testing.T) {
	k := &Keyring{
		KeyringPayload: "sensitive data",
	}

	t.Run("puzzle generation and solving", func(t *testing.T) {
		k.GenerateTimeLockedPuzzle(1)
		require.NotNil(t, k.TLP)
		require.NotEmpty(t, k.TLP.Puzzle)
		require.NotEmpty(t, k.TLP.Payload)
		require.NotEmpty(t, k.TLP.Iter)
		require.NotEmpty(t, k.TLP.N)

		// Reconstruct parameters with error checking
		puzzleInt, ok := new(big.Int).SetString(k.TLP.Puzzle, 10)
		require.True(t, ok, "Failed to parse puzzle")

		iterations, ok := new(big.Int).SetString(k.TLP.Iter, 10)
		require.True(t, ok, "Failed to parse iterations")

		n, ok := new(big.Int).SetString(k.TLP.N, 10)
		require.True(t, ok, "Failed to parse modulus")

		// Break the puzzle
		start := time.Now()
		base := big.NewInt(2)
		for i := big.NewInt(0); i.Cmp(iterations) < 0; i.Add(i, big.NewInt(1)) {
			base.Exp(base, big.NewInt(2), n)
		}

		// Retrieve AES key
		key := new(big.Int).Sub(puzzleInt, base)
		key.Mod(key, n)

		// Decrypt payload
		result, err := AESToPayload(k.TLP.Payload, key)
		require.NoError(t, err)
		require.Equal(t, k.KeyringPayload, result)

		t.Logf("Puzzle solved in %v", time.Since(start))
	})

	t.Run("invalid parameters", func(t *testing.T) {
		k := &Keyring{KeyringPayload: "test"}
		k.GenerateTimeLockedPuzzle(1)

		// Parse original puzzle
		originalPuzzle, ok := new(big.Int).SetString(k.TLP.Puzzle, 10)
		require.True(t, ok, "Invalid puzzle format")

		// Create corrupted puzzle (already a *big.Int, no need to parse)
		corruptedPuzzle := new(big.Int).Add(originalPuzzle, big.NewInt(1))

		// Parse other components
		n, ok := new(big.Int).SetString(k.TLP.N, 10)
		require.True(t, ok, "Invalid modulus format")

		iterations, ok := new(big.Int).SetString(k.TLP.Iter, 10)
		require.True(t, ok, "Invalid iteration format")

		// Break puzzle with wrong parameters
		base := big.NewInt(2)
		for i := big.NewInt(0); i.Cmp(iterations) < 0; i.Add(i, big.NewInt(1)) {
			base.Exp(base, big.NewInt(2), n)
		}

		// Use corrupted puzzle directly (already a *big.Int)
		key := new(big.Int).Sub(corruptedPuzzle, base)
		key.Mod(key, n)

		_, err := AESToPayload(k.TLP.Payload, key)
		require.Error(t, err, "Should fail with corrupted puzzle")
	})
}

func TestPuzzleProperties(t *testing.T) {
	k := &Keyring{KeyringPayload: "test payload"}
	k.GenerateTimeLockedPuzzle(1)

	t.Run("prime properties", func(t *testing.T) {
		n, _ := new(big.Int).SetString(k.TLP.N, 10)
		require.Greater(t, n.BitLen(), 2048, "modulus too small")
	})

	t.Run("iteration calculation", func(t *testing.T) {
		iterations, _ := new(big.Int).SetString(k.TLP.Iter, 10)
		require.Greater(t, iterations.Sign(), 0, "invalid iteration count")
	})

	t.Run("payload integrity", func(t *testing.T) {
		_, err := base64.StdEncoding.DecodeString(k.TLP.Payload)
		require.NoError(t, err, "invalid base64 payload")
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("empty keyring payload", func(t *testing.T) {
		k := &Keyring{KeyringPayload: ""}
		require.NotPanics(t, func() {
			k.GenerateTimeLockedPuzzle(1)
		})
	})

	t.Run("invalid AES parameters", func(t *testing.T) {
		_, err := AESToPayload("invalid", big.NewInt(0))
		require.Error(t, err)
	})
}
