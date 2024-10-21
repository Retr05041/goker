package main

import (
	"fmt"
	"flag"
	"os"
	"goker/internal/p2p"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"math/big"
	"goker/internal/sra"
)

func main() {
	mycyphertest()
}

func connect() {
	sourcePort := flag.Int("sp", 0, "Source port number")
	dest := flag.String("d", "", "Destination multiaddr string")
	help := flag.Bool("help", false, "Display help")

	flag.Parse()

	if *help {
		fmt.Printf("Basic p2p networking client\n\n")
		fmt.Println("Usage: Run './goker -sp <SOURCE_PORT>' where <SOURCE_PORT> can be any port number.")
		fmt.Println("Now run './goker -d <MULTIADDR>' where <MULTIADDR> is multiaddress of previous listener host.")

		os.Exit(0)
	}

	p2p.Init(sourcePort, dest)
}

func mycyphertest() {
	// generate a shared p and q
	p, err := sra.GenerateLargePrime(8)
	if err != nil {
		fmt.Println(err)
	}
	q, err := sra.GenerateLargePrime(8)
	if err != nil {
		fmt.Println(err)
	}

    // Array to hold public keys
    var publicKeys []*big.Int

	alicePublicKey, alicePrivateKey, aliceModN, err := sra.GenerateKeys(nil, p, q)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Alice's keys have been generated")

	publicKeys = append(publicKeys, alicePublicKey)

	bobPublicKey, bobPrivateKey, bobModN, err := sra.GenerateKeys(publicKeys, p, q)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Bob's keys have been generated")

	// Example message
	message := []byte("X")

	// Step 2: Hash the message
	hash := sha256.Sum256(message)

	// Step 3: Encrypt the hash with Alice's public key
	aliceCipher := new(big.Int).Exp(new(big.Int).SetBytes(hash[:]), alicePublicKey, aliceModN)

	// Step 4: Encrypt the already encrypted hash with Bob's public key
	bobCipher := new(big.Int).Exp(aliceCipher, bobPublicKey, bobModN)

	// Step 5: Alice decrypts the message with her private key
	aliceDecrypted := new(big.Int).Exp(bobCipher, alicePrivateKey, aliceModN)
	
	// Step 6: Bob decrypts the message with his private key
	finalHash := new(big.Int).Exp(aliceDecrypted, bobPrivateKey, bobModN)

	// Convert back to bytes
	decryptedHash := finalHash.Bytes()

	// Pad the decrypted hash to match the original hash size
	expectedHash := make([]byte, sha256.Size)
	copy(expectedHash[sha256.Size-len(decryptedHash):], decryptedHash)

	// Verify that the decrypted hash matches the original hash
	fmt.Printf("Orignal message: %s\n", message)
	fmt.Printf("Original hash: %x\n", hash)
	fmt.Println("---")
	fmt.Printf("Alice encyrpts the hash: %x\n", aliceCipher)
	fmt.Printf("Bob encyrpts Alice's hash: %x\n", bobCipher)
	fmt.Println("---")
	fmt.Printf("Alice's decrypts her hash: %x\n", aliceDecrypted)
	fmt.Printf("Bob decrypts his hash: %x\n", expectedHash)
	fmt.Println("---")
	fmt.Printf("Message matches: %t\n", string(expectedHash) == string(hash[:]))
}


func cypherTest() {
	// Step 1: Generate RSA keys for Alice and Bob
	aliceKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	bobKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	// Example message
	message := []byte("Hello, Bob!")

	// Step 2: Hash the message
	hash := sha256.Sum256(message)

	// Step 3: Encrypt the hash with Alice's public key
	aliceCipher := new(big.Int).Exp(new(big.Int).SetBytes(hash[:]), big.NewInt(int64(aliceKey.PublicKey.E)), aliceKey.PublicKey.N)

	// Step 4: Encrypt the already encrypted hash with Bob's public key
	bobCipher := new(big.Int).Exp(aliceCipher, big.NewInt(int64(bobKey.PublicKey.E)), bobKey.PublicKey.N)

	// Step 5: Alice decrypts the message with her private key
	aliceDecrypted := new(big.Int).Exp(bobCipher, aliceKey.D, aliceKey.PublicKey.N)
	
	// Step 6: Bob decrypts the message with his private key
	finalHash := new(big.Int).Exp(aliceDecrypted, bobKey.D, bobKey.PublicKey.N)

	// Convert back to bytes
	decryptedHash := finalHash.Bytes()

	// Pad the decrypted hash to match the original hash size
	expectedHash := make([]byte, sha256.Size)
	copy(expectedHash[sha256.Size-len(decryptedHash):], decryptedHash)

	// Verify that the decrypted hash matches the original hash
	fmt.Printf("Orignal message: %s\n", message)
	fmt.Printf("Original hash: %x\n", hash)
	fmt.Println("---")
	fmt.Printf("Alice encyrpts the hash: %x\n", aliceCipher)
	fmt.Printf("Bob encyrpts Alice's hash: %x\n", bobCipher)
	fmt.Println("---")
	fmt.Printf("Alice's decrypts her hash: %x\n", aliceDecrypted)
	fmt.Printf("Bob decrypts his hash: %x\n", expectedHash)
	fmt.Println("---")
	fmt.Printf("Message matches: %t\n", string(expectedHash) == string(hash[:]))
}


func cypherTest3() {
	// Step 1: Generate RSA keys for Alice and Bob
	aliceKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	bobKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	// Example message
	message := []byte("Hello, Bob!")

	// Step 2: Hash the message
	hash := sha256.Sum256(message)

	// Step 3: Encrypt the hash with Alice's public key
	aliceCipher := new(big.Int).Exp(new(big.Int).SetBytes(hash[:]), big.NewInt(int64(aliceKey.PublicKey.E)), aliceKey.PublicKey.N)

	// Step 4: Encrypt the already encrypted hash with Bob's public key
	bobCipher := new(big.Int).Exp(aliceCipher, big.NewInt(int64(bobKey.PublicKey.E)), bobKey.PublicKey.N)

	// Step 5: Bob decrypts the message with his private key
	bobDecrypted := new(big.Int).Exp(bobCipher, bobKey.D, bobKey.PublicKey.N)
	
	// Step 6: Alice decrypts the message with her private key
	finalHash := new(big.Int).Exp(bobDecrypted, aliceKey.D, aliceKey.PublicKey.N)

	// Convert back to bytes
	decryptedHash := finalHash.Bytes()

	// Pad the decrypted hash to match the original hash size
	expectedHash := make([]byte, sha256.Size)
	copy(expectedHash[sha256.Size-len(decryptedHash):], decryptedHash)

	// Verify that the decrypted hash matches the original hash
	fmt.Printf("Orignal message: %s\n", message)
	fmt.Printf("Original hash: %x\n", hash)
	fmt.Println("---")
	fmt.Printf("Alice encyrpts the hash: %x\n", aliceCipher)
	fmt.Printf("Bob encyrpts Alice's hash: %x\n", bobCipher)
	fmt.Println("---")
	fmt.Printf("Bob decrypts his hash: %x\n", bobDecrypted)
	fmt.Printf("Alice decrypts her hash: %x\n", expectedHash)
	fmt.Println("---")
	fmt.Printf("Message matches: %t\n", string(expectedHash) == string(hash[:]))
}


func cypherTest2() {
	// Step 1: Generate RSA keys for Alice
	aliceKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	// Original message (make sure it fits within the size limit)
	message := []byte("Hello, Bob! This is a longer message that may need to be truncated.")

	// Step 2: Check the max size for the RSA key (minus padding)
	maxSize := (aliceKey.PublicKey.N.BitLen()/8) - 11 // Adjust for padding (e.g., OAEP)

	if len(message) > maxSize {
		// Truncate the message to fit
		message = message[:maxSize]
		fmt.Printf("Truncated message: %s\n", message)
	}

	// Step 3: Encrypt the modified message with Alice's public key
	encryptedMessage, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &aliceKey.PublicKey, message, nil)
	if err != nil {
		panic(err)
	}

	if len(encryptedMessage) > maxSize {
		fmt.Printf("ALICES KEY IS TOO LONG TO BE ENCRYPTED AGAIN: %s\n", encryptedMessage)
	}

	// Step 4: Decrypt the message with Alice's private key
	decryptedMessage, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, aliceKey, encryptedMessage, nil)
	if err != nil {
		panic(err)
	}

	// Display results
	fmt.Printf("Original Message: %s\n", message)
	fmt.Printf("Decrypted Message: %s\n", decryptedMessage)
}
