package main

import (
	"fmt"
	"flag"
	"os"
	"goker/internal/p2p"
	"goker/internal/sra"
)

func main() {
	cypherTest()
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


func cypherTest() {
	// Generate keys
	e1, d1, n1, err := crypto.GenerateKeys()
	if err != nil {
		fmt.Println("Error generating keys:", err)
		return
	}
	fmt.Println("KEYS GENERATED")

	// Original message
	message := "Hello, World." 

	// Encrypt message with e1 and e2
	c1 := crypto.Encrypt(message, e1, n1)
	fmt.Println("MESSAGE ENCRYPTED")

	// Decrypt both results
	decrypted1 := crypto.Decrypt(c1, d1, n1)
	fmt.Println("MESSAGE DECRYPTED")

	// Show results
	fmt.Println("Original Message:", message)
	fmt.Println("Ciphertext 1:", c1)
	fmt.Println("Decrypted Result:", decrypted1)

	// Check if the decrypted results match the original message
	if decrypted1 == message {
		fmt.Println("Decrypted Result 1 matches the original message.")
	} else {
		fmt.Println("Decrypted Result 1 does NOT match the original message.")
	}
}
