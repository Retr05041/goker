package main

import (
	"fmt"
	"flag"
	"os"
	"goker/internal/p2p"
)

func main() {
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
