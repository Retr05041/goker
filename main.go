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
	message := "hello"
    fmt.Println("Original Message: ", message)

	publicKey, privateKey, modN, err := crypto.GenerateKeys()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Keys generated")

    cipherText := crypto.Encrypt(message, publicKey, modN)
    fmt.Println("Ciphertext: ", cipherText.String())

    decryptedMessage := crypto.Decrypt(cipherText, privateKey, modN)
    fmt.Println("Decrypted Result: ", decryptedMessage)

    if decryptedMessage == message {
        fmt.Println("Decrypted Result matches the original message.")
    } else {
        fmt.Println("Decrypted Result does NOT match the original message.")
    }
}
