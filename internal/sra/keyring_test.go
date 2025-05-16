package sra

import (
	"math/big"
	"strings"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestKeyringSigningOperations(t *testing.T) {
	t.Run("generate and export keys", func(t *testing.T) {
		k := &Keyring{}
		err := k.GenerateSigningKeys()
		require.NoError(t, err)
		require.NotNil(t, k.signingPrivKey)
		require.NotNil(t, k.signingPubKey)

		pem, err := k.ExportPublicKey()
		require.NoError(t, err)
		require.Contains(t, pem, "PUBLIC KEY")
	})

	t.Run("sign and verify message", func(t *testing.T) {
		k := &Keyring{}
		k.GenerateSigningKeys()
		peerID := peer.ID("test-peer")

		// Create and register peer key
		peerK := &Keyring{}
		peerK.GenerateSigningKeys()
		pem, _ := peerK.ExportPublicKey()
		k.SetPeerPublicKey(peerID, pem)

		msg := "test message"
		sig, err := peerK.SignMessage(msg)
		require.NoError(t, err)

		valid := k.VerifySignature(peerID, msg, sig)
		require.True(t, valid)

		t.Run("invalid signature", func(t *testing.T) {
			valid := k.VerifySignature(peerID, msg, "invalid"+sig)
			require.False(t, valid)
		})

		t.Run("wrong message", func(t *testing.T) {
			valid := k.VerifySignature(peerID, "wrong message", sig)
			require.False(t, valid)
		})
	})
}

func TestPrimeOperations(t *testing.T) {
	t.Run("generate and set primes", func(t *testing.T) {
		k := &Keyring{}
		k.GeneratePQ()
		require.NotNil(t, k.sharedP)
		require.NotNil(t, k.sharedQ)
		require.True(t, k.sharedP.ProbablyPrime(20))
		require.True(t, k.sharedQ.ProbablyPrime(20))
		require.NotEqual(t, k.sharedP, k.sharedQ)

		// Test serialization
		pqStr := k.GetPQString()
		parts := strings.Split(strings.TrimSpace(pqStr), "\n")
		require.Len(t, parts, 2)

		// Test deserialization
		newK := &Keyring{}
		newK.SetPQ(parts[0], parts[1])
		require.Equal(t, k.sharedP.String(), newK.sharedP.String())
		require.Equal(t, k.sharedQ.String(), newK.sharedQ.String())
	})
}

func TestSRAOperations(t *testing.T) {
	k := &Keyring{}
	k.GeneratePQ()
	require.NoError(t, k.GenerateKeys())

	t.Run("key generation", func(t *testing.T) {
		require.NotNil(t, k.globalPublicKey)
		require.NotNil(t, k.globalPrivateKey)
		require.NotNil(t, k.globalN)
		require.NotNil(t, k.globalPHI)

		// Verify n = p*q
		expectedN := new(big.Int).Mul(k.sharedP, k.sharedQ)
		require.Equal(t, 0, k.globalN.Cmp(expectedN))

		// Verify phi = (p-1)(q-1)
		pMinus1 := new(big.Int).Sub(k.sharedP, big.NewInt(1))
		qMinus1 := new(big.Int).Sub(k.sharedQ, big.NewInt(1))
		expectedPhi := new(big.Int).Mul(pMinus1, qMinus1)
		require.Equal(t, 0, k.globalPHI.Cmp(expectedPhi))

		// Verify e*d ≡ 1 mod phi
		product := new(big.Int).Mul(k.globalPublicKey, k.globalPrivateKey)
		modResult := new(big.Int).Mod(product, k.globalPHI)
		require.Equal(t, 0, modResult.Cmp(big.NewInt(1)))
	})

	t.Run("encryption/decryption", func(t *testing.T) {
		original := big.NewInt(123456789)
		encrypted := new(big.Int).Set(original)
		k.EncryptWithGlobalKeys(encrypted)

		require.NotEqual(t, 0, original.Cmp(encrypted))

		decrypted := new(big.Int).Set(encrypted)
		k.DecryptWithGlobalKeys(decrypted)

		require.Equal(t, 0, original.Cmp(decrypted))
	})

	t.Run("generate keys without primes", func(t *testing.T) {
		k := &Keyring{}
		err := k.GenerateKeys()
		require.Error(t, err)
	})
}

func TestPeerKeyManagement(t *testing.T) {
	t.Run("set valid peer key", func(t *testing.T) {
		k := &Keyring{}
		peerK := &Keyring{}
		peerK.GenerateSigningKeys()
		pem, _ := peerK.ExportPublicKey()

		peerID := peer.ID("test-peer")
		k.SetPeerPublicKey(peerID, pem)

		storedKey, exists := k.Otherskeys[peerID]
		require.True(t, exists)
		require.Equal(t, peerK.signingPubKey.E, storedKey.E)
		require.Equal(t, peerK.signingPubKey.N, storedKey.N)
	})

	t.Run("set invalid peer key", func(t *testing.T) {
		k := &Keyring{}
		peerID := peer.ID("test-peer")
		k.SetPeerPublicKey(peerID, "invalid-pem")

		_, exists := k.Otherskeys[peerID]
		require.False(t, exists)
	})
}
