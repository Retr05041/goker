package p2p

import (
	"bufio"
	"context"
	"fmt"
	"goker/internal/channelmanager"
	"io"
	"log"
	"strings"

	"github.com/libp2p/go-libp2p/core/network"
)

type Command interface {
	Execute(peer *GokerPeer)
	Respond(peer *GokerPeer, sendingStream network.Stream)
}

// Execute a command
func (p *GokerPeer) ExecuteCommand(command Command) {
	command.Execute(p)
}

// Respond to a command
func (p *GokerPeer) RespondToCommand(command Command, stream network.Stream) {
	command.Respond(p, stream)
}

// Handle incoming streams
// If it's a new host in the network, assume it's waiting for the peer list. Else, Assume it's a command
func (p *GokerPeer) handleStream(stream network.Stream) {
	defer stream.Close()
	fmt.Println("New stream detected... getting command.")

	// Read incoming command
	var message strings.Builder
	reader := bufio.NewReader(stream)
	for {
		// Read line by line
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Println("handleStream: error reading from stream: %w", err)
		}

		// Check for the end marker
		if strings.TrimSpace(line) == "\\END" {
			break
		}

		// Append the line to the payload
		message.WriteString(line)
	}

	cleanedMessage := strings.TrimSpace(message.String())

	// Split the command and the payload
	parts := strings.SplitN(cleanedMessage, " ", 2)
	command := parts[0]
	fmt.Println(command)
	var payload string
	if len(parts) > 1 {
		payload = parts[1]
	}

	// Process the command based on the message
	switch command {
	case "CMDgetpeers": // Send peerlist to just this stream
		fmt.Println("Recieved peer list request")
		peerList := p.getPeerList()
		_, err := stream.Write([]byte(peerList)) // Ensure the response is newline-terminated
		if err != nil {
			log.Printf("Failed to send peer list: %v", err)
		}
	case "CMDpqrequest":
		fmt.Println("Recieved PQ Request")
		p.RespondToCommand(&PQRequestCommand{}, stream)
	case "CMDprotocolFS": // First step of Protocol
		fmt.Println("Recieved protocols first step command")
		p.deck.SetDeck(payload)
		p.RespondToCommand(&ProtocolFirstStepCommand{}, stream)
	case "CMDprotocolSS": // Second step of Protocol
		fmt.Println("Recieved protocols second step command")
		p.deck.SetDeck(payload)
		p.RespondToCommand(&ProtocolFirstStepCommand{}, stream)
	case "CMDstartround":
		fmt.Println("Recieved start round command")
		p.RespondToCommand(&StartRoundCommand{}, stream)
	default:
		log.Printf("Unknown Response Recieved: %s\n", cleanedMessage)
	}
}


/////////////////////////////////////////////////////////////////////////////////////////////////////

// Request the shared P and Q every peer should have in the network - all peers will request this from the host each round
type PQRequestCommand struct{}

func (pq *PQRequestCommand) Execute(peer *GokerPeer) {
	// Create a new stream to the peer
	stream, err := peer.thisHost.NewStream(context.Background(), peer.sessionHost, protocolID)
	if err != nil {
		log.Printf("PQRequest: Failed to create stream to host %s: %v\n", peer.sessionHost, err)
		return
	}
	defer stream.Close()

	_, err = stream.Write([]byte("CMDpqrequest\n\\END\n"))
	if err != nil {
		log.Printf("PQRequest: Failed to send command to host %s: %v\n", peer.sessionHost, err)
	} else {
		fmt.Printf("PQRequest: Sent command to host %s\n", peer.sessionHost)
	}

	// Read the response - we do this as they won't be sending an actual *command* back, just some text
	responseBytes, err := io.ReadAll(stream)
	if err != nil {
		log.Printf("PQRequest: Failed to read response from host %s: %v\n", peer.sessionHost, err)
	} else {
		pq := strings.Split(string(responseBytes), "\n")
		fmt.Printf("PQRequest: Received response from host: %s P: %s -- Q: %s\n", peer.sessionHost, pq[0], pq[1])
		peer.keyring.SetPQ(pq[0], pq[1]) // Set this servers p and q
		peer.keyring.GenerateKeys()
	}
}

// Called on a 'CMDpqrequest' command call
func (pq *PQRequestCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	PQData := peer.keyring.GetPQString()
	_, err := sendingStream.Write([]byte(PQData))
	if err != nil {
		log.Printf("PQRequestResponse: Failed to send PQ to peer %s: %v\n", sendingStream.Conn().RemotePeer(), err)
	} else {
		fmt.Printf("PQRequestResponse: Sent PQ to peer %s\n", sendingStream.Conn().RemotePeer())
	}
}

/////////////////////////////////////////////////////////////////////////////////////////////////////

// / ### PROTOCOL ###
type ProtocolFirstStepCommand struct{}

// Send deck to every peer, allow them to shuffle and encrypt the deck
func (sp *ProtocolFirstStepCommand) Execute(peer *GokerPeer) {
	peer.EncryptAllWithGlobalKeys()
	peer.deck.ShuffleRoundDeck()

	peer.peerListMutex.Lock()
	defer peer.peerListMutex.Unlock()

	for peerID := range peer.peerList {
		if peerID == peer.thisHost.ID() {
			continue
		}

		// Create a new stream to the peer
		stream, err := peer.thisHost.NewStream(context.Background(), peerID, protocolID)
		if err != nil {
			log.Printf("ProtocolFirstStep: Failed to create stream to peer %s: %v\n", peerID, err)
			continue
		}
		defer stream.Close()

		// Send the deck
		_, err = stream.Write([]byte(fmt.Sprintf("CMDprotocolFS %s\n\\END\n", peer.deck.GenerateDeckPayload())))
		if err != nil {
			log.Printf("ProtocolFirstStep: Failed to send deck to peer %s: %v\n", peerID, err)
		} else {
			fmt.Printf("ProtocolFirstStep: Sent deck to peer %s\n", peerID)
		}

		// Get the response (the new deck)
		responseBytes, err := io.ReadAll(stream)
		if err != nil {
			log.Printf("ProtocolFirstStep: Failed to read response from host %s: %v\n", peer.sessionHost, err)
		} else {
			peer.deck.SetDeck(string(responseBytes))
			fmt.Printf("ProtocolFirstStep: Received response from peer %s\n", peerID)
		}

	}
}

// Respond to a protocol's first command - Encrypt with global keys, shuffle, then send back - when this is called a new deck should be set already
func (sp *ProtocolFirstStepCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	// Encrypt the deck with your global keys, shuffle it, then create a new payload to send back
	peer.EncryptAllWithGlobalKeys()
	peer.deck.ShuffleRoundDeck()
	processedDeck := peer.deck.GenerateDeckPayload()

	// Send the updated deck back to the sender
	_, err := sendingStream.Write([]byte(fmt.Sprintf("%s\\END\n", processedDeck)))
	if err != nil {
		log.Printf("StartProtocol Respond: Failed to send response: %v\n", err)
	}
}

type ProtocolSecondStepCommand struct{}

// For getting others to encrypt with their variation keys
func (sp *ProtocolSecondStepCommand) Execute(peer *GokerPeer) {
	// First, the host of the game (the one initialing the protocols steps) will encrypt with variations
	peer.DecryptAllWithGlobalKeys() // Decrypt global keys
	peer.EncryptAllWithVariation() // Add encryption to every card

	// Time to get the peers to do the same
	peer.peerListMutex.Lock()
	defer peer.peerListMutex.Unlock()

	for peerID := range peer.peerList {
		if peerID == peer.thisHost.ID() {
			continue
		}

		// Create a new stream to the peer
		stream, err := peer.thisHost.NewStream(context.Background(), peerID, protocolID)
		if err != nil {
			log.Printf("ProtocolSecondStep: Failed to create stream to peer %s: %v\n", peerID, err)
			continue
		}
		defer stream.Close()

		// Send the deck
		_, err = stream.Write([]byte(fmt.Sprintf("CMDprotocolSS %s\n\\END\n", peer.deck.GenerateDeckPayload())))
		if err != nil {
			log.Printf("ProtocolSecondStep: Failed to send deck to peer %s: %v\n", peerID, err)
		} else {
			fmt.Printf("ProtocolSecondStep: Sent deck to peer %s\n", peerID)
		}

		// Get the response (the new deck)
		responseBytes, err := io.ReadAll(stream)
		if err != nil {
			log.Printf("ProtocolSecondStep: Failed to read response from host %s: %v\n", peer.sessionHost, err)
		} else {
			peer.deck.SetDeck(string(responseBytes)) // Setting the deck without changing it as no shuffling was done
			fmt.Printf("ProtocolSecondStep: Received response from peer %s\n", peerID)
		}
	}
}

// Respond to the protocols second command - Decrypt global keys, encrypt with variation, then send back- when this is called a new deck should be set already
func (sp *ProtocolSecondStepCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	peer.DecryptAllWithGlobalKeys()
	peer.EncryptAllWithVariation()
	processedDeck := peer.deck.GenerateDeckPayload()

	// Send the updated deck back to the sender
	_, err := sendingStream.Write([]byte(fmt.Sprintf("%s\\END\n", processedDeck)))
	if err != nil {
		log.Printf("StartProtocol Respond: Failed to send response: %v\n", err)
	}
}

/////////////////////////////////////////////////////////////////////////////////////////////////////

type StartRoundCommand struct{}

func (sg *StartRoundCommand) Execute(peer *GokerPeer) {
		
	peer.peerListMutex.Lock()
	defer peer.peerListMutex.Unlock()

	for peerID := range peer.peerList {
		if peerID == peer.thisHost.ID() {
			continue
		}

		// Create a new stream to the peer
		stream, err := peer.thisHost.NewStream(context.Background(), peerID, protocolID)
		if err != nil {
			log.Printf("StarRoundCommand: Failed to create stream to peer %s: %v\n", peerID, err)
			continue
		}
		defer stream.Close()

		// Send the deck
		_, err = stream.Write([]byte(fmt.Sprintf("CMDstartround\n\\END\n")))
		if err != nil {
			log.Printf("StartRoundCommand: Failed to send command to peer %s: %v\n", peerID, err)
		} else {
			fmt.Printf("StartRoundCommand: Sent command to peer %s\n", peerID)
		}
	}
}

func (sg *StartRoundCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	channelmanager.FNET_StartRoundChan <- true
}
