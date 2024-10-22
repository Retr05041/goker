package main

import (
	"fmt"
	"flag"
	"os"
	"goker/internal/p2p"
	"crypto/sha256"
	"math/big"
	"goker/internal/sra"
)

func main() {
	cyphertest()
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

func cyphertest() {
	// generate a shared p and q
	p, err := sra.GenerateLargePrime(2048)
	if err != nil {
		fmt.Println(err)
	}
	q, err := sra.GenerateLargePrime(2048)
	if err != nil {
		fmt.Println(err)
	}

	alicePublicKey, alicePrivateKey, aliceModN, err := sra.GenerateKeys(p, q)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Alice's keys have been generated")

	bobPublicKey, bobPrivateKey, bobModN, err := sra.GenerateKeys(p, q)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Bob's keys have been generated")

	// Example message
	message := []byte("Hello, World.")

	// Step 2: Hash the message
	hash := sha256.Sum256(message)
	hashBigInt := new(big.Int).SetBytes(hash[:])

	// Step 3: Encrypt the hash with Alice's public key
	aliceCipher := sra.Encrypt(hashBigInt, alicePublicKey, aliceModN)

	// Step 4: Encrypt the already encrypted hash with Bob's public key
	bobCipher := sra.Encrypt(aliceCipher, bobPublicKey, bobModN)

	// Step 5: Alice decrypts the message with her private key
	aliceDecrypted := sra.Decrypt(bobCipher, alicePrivateKey, aliceModN)
	
	// Step 6: Bob decrypts the message with his private key
	finalHash := sra.Decrypt(aliceDecrypted, bobPrivateKey, bobModN)

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
