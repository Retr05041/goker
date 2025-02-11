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

// Uses Command Design Pattern
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

// Handle incoming streams (should be commands only)
func (p *GokerPeer) handleStream(stream network.Stream) {
	defer stream.Close()

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
		p.RespondToCommand(&GetPeerListCommand{}, stream)
	case "CMDnicknamerequest":
		log.Println("Recieved nickname request command")
		p.RespondToCommand(&NicknameRequestCommand{}, stream)
	case "CMDsendpq": // No need to build a response for this
		log.Println("Recieved Send P & Q command")
		pq := strings.Split(payload, "\n")
		p.Keyring.SetPQ(pq[0], pq[1]) // Set P and Q
		p.Keyring.GenerateKeys()      // Create Keys
	case "CMDprotocolFS": // First step of Protocol
		log.Println("Recieved protocols first step command")
		p.Deck.SetDeck(payload)
		p.RespondToCommand(&ProtocolFirstStepCommand{}, stream)
	case "CMDprotocolSS": // Second step of Protocol
		log.Println("Recieved protocols second step command")
		p.Deck.SetDeck(payload)
		p.RespondToCommand(&ProtocolFirstStepCommand{}, stream)
		p.SetHands() // Make sure at the end of the second step of the protocol to set everyones hands for that round
	case "CMDinittable":
		log.Println("Recieved init table command")
		// Populate the state with peoples info
		p.SetTurnOrderWithLobby()
		// Set the rest of the state
		p.gameState.FreshStateFromPayload(payload)
		channelmanager.TGUI_PlayerInfo <- p.gameState.GetPlayerInfo() // Update GUI cards
		p.RespondToCommand(&InitTableCommand{}, stream)
	case "CMDrequesthand": // No payload, simply a peer requesting their card keys - the peerlist should all be signed before this ever gets called by a peer
		log.Println("Recieved request hand command")
		keyPayload := p.GetKeyPayloadForPlayersHand(stream.Conn().RemotePeer())
		fmt.Println(keyPayload)
		_, err := stream.Write([]byte(keyPayload))
		if err != nil {
			log.Printf("RequestHand: Failed to send keys to peer %s: %v\n", stream.Conn().RemotePeer(), err)
		} else {
			fmt.Printf("RequestHand: Sent keys to peer %s\n", stream.Conn().RemotePeer())
		}
	default:
		log.Printf("Unknown Command Recieved: %s\n", command)
	}
}

//////////////////////////////////////////// NEGOTIATION ///////////////////////////////////////////////////

type GetPeerListCommand struct{}

func (gpl *GetPeerListCommand) Execute(peer *GokerPeer) {
	stream, err := peer.ThisHost.NewStream(context.Background(), peer.sessionHost.ID, protocolID)
	if err != nil {
		log.Printf("GetPeerListCommand: Failed to create stream to host %s: %v\n", peer.sessionHost, err)
		return
	}
	defer stream.Close()

	// Aquire and set peerlist
	_, err = stream.Write([]byte("CMDgetpeers\n\\END\n"))
	if err != nil {
		log.Fatalf("GetPeerListCommand: Failed to send CMDgetpeers command: %v", err)
	}

	peerListBytes, err := io.ReadAll(stream)
	if err != nil {
		log.Fatalf("GetPeerListCommand: Failed to read peer list: %v", err)
	}

	peer.setPeerListAndConnect(string(peerListBytes))
	log.Println("GetPeerListCommand: Received and set peerlist.")
}

func (gpl *GetPeerListCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	_, err := sendingStream.Write([]byte(peer.getPeerList()))
	if err != nil {
		log.Printf("GetPeerlistCommand: Failed to send peer list: %v", err)
	}
}

type NicknameRequestCommand struct{} // Loops through every peer in the peer list, and for any we don't have nicknames for, request one

func (nr *NicknameRequestCommand) Execute(peer *GokerPeer) {
	peer.peerListMutex.Lock()
	defer peer.peerListMutex.Unlock()

	for _, peerInfo := range peer.peerList {
		if peerInfo.ID == peer.ThisHost.ID() { // If it's us
			continue
		}
		if peer.gameState.PlayerExists(peerInfo.ID) { // if the player already exists, then obviously we don't need their nickname
			continue
		}

		// Create a new stream to the peer
		stream, err := peer.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("NicknameRequest: Failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		_, err = stream.Write([]byte("CMDnicknamerequest\n\\END\n"))
		if err != nil {
			log.Printf("NicknameRequest: Failed to send command to peer%s: %v\n", peerInfo.ID, err)
		} else {
			fmt.Printf("NicknameRequest: Sent command to peer%s\n", peerInfo.ID)
		}

		// Read the response - we do this as they won't be sending an actual *command* back, just some text
		responseBytes, err := io.ReadAll(stream)
		if err != nil {
			log.Printf("NicknameRequest: Failed to read response from host %s: %v\n", peerInfo.ID, err)
		} else {
			peerNickname := strings.Split(string(responseBytes), "\n")
			fmt.Printf("NicknameRequest: Received response from peer: %s -- Nickname: %s\n", peerInfo.ID, peerNickname[0])
			peer.gameState.AddPeerToState(peerInfo.ID, peerNickname[0]) // Finally add peer to gamestate
		}
	}
}

func (nr *NicknameRequestCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	myNickname := peer.gameState.GetNickname(peer.ThisHost.ID())
	_, err := sendingStream.Write([]byte(myNickname))
	if err != nil {
		log.Printf("NicknameRequest: Failed to send nickname to peer %s: %v\n", sendingStream.Conn().RemotePeer(), err)
	} else {
		fmt.Printf("NicknameRequest: Sent nickname to peer %s\n", sendingStream.Conn().RemotePeer())
	}
}

////////////////////////////////////////// KEYRING //////////////////////////////////////////////////////

// Send P and Q to all peers in the network so they may generate their GenerateKeys
// TODO: Have all peers validate P and Q's bit length and that they are prime then sign and send to next
type SendPQCommand struct{}

func (pq *SendPQCommand) Execute(peer *GokerPeer) {
	peer.peerListMutex.Lock()
	defer peer.peerListMutex.Unlock()

	for _, peerInfo := range peer.peerList {
		if peerInfo.ID == peer.ThisHost.ID() {
			continue
		}
		// Create a new stream to the peer
		stream, err := peer.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("SendPQCommand: Failed to create stream to peer %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		PQData := peer.Keyring.GetPQString()
		_, err = stream.Write([]byte(fmt.Sprintf("CMDsendpq %s\n\\END\n", PQData)))
		if err != nil {
			log.Printf("SendPQCommand: Failed to send command to host %s: %v\n", peerInfo.ID, err)
		} else {
			fmt.Printf("SendPQCommand: Sent command to peer %s\n", peerInfo.ID)
		}
	}
}

func (pq *SendPQCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {}

///////////////////////////////////////////// PROTOCOL ///////////////////////////////////////////////////

type ProtocolFirstStepCommand struct{}

// Send deck to every peer, allow them to shuffle and encrypt the deck
func (sp *ProtocolFirstStepCommand) Execute(peer *GokerPeer) {
	peer.EncryptAllWithGlobalKeys()
	peer.Deck.ShuffleRoundDeck()

	peer.peerListMutex.Lock()
	defer peer.peerListMutex.Unlock()

	for _, peerInfo := range peer.peerList {
		if peerInfo.ID == peer.ThisHost.ID() {
			continue
		}

		// Create a new stream to the peer
		stream, err := peer.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("ProtocolFirstStep: Failed to create stream to peer %s: %v\n", peerInfo.ID, err)
			continue
		}
		defer stream.Close()

		// Send the deck
		_, err = stream.Write([]byte(fmt.Sprintf("CMDprotocolFS %s\n\\END\n", peer.Deck.GenerateDeckPayload())))
		if err != nil {
			log.Printf("ProtocolFirstStep: Failed to send deck to peer %s: %v\n", peerInfo.ID, err)
		} else {
			fmt.Printf("ProtocolFirstStep: Sent deck to peer %s\n", peerInfo.ID)
		}

		// Get the response (the new deck)
		responseBytes, err := io.ReadAll(stream)
		if err != nil {
			log.Printf("ProtocolFirstStep: Failed to read response from host %s: %v\n", peer.sessionHost, err)
		} else {
			peer.Deck.SetDeck(string(responseBytes))
			fmt.Printf("ProtocolFirstStep: Received response from peer %s\n", peerInfo.ID)
		}

	}
}

// Respond to a protocol's first command - Encrypt with global keys, shuffle, then send back - when this is called a new deck should be set already
func (sp *ProtocolFirstStepCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	// Encrypt the deck with your global keys, shuffle it, then create a new payload to send back
	peer.EncryptAllWithGlobalKeys()
	peer.Deck.ShuffleRoundDeck()
	processedDeck := peer.Deck.GenerateDeckPayload()

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
	peer.EncryptAllWithVariation()  // Add encryption to every card

	// Time to get the peers to do the same
	peer.peerListMutex.Lock()
	defer peer.peerListMutex.Unlock()

	for _, peerInfo := range peer.peerList {
		if peerInfo.ID == peer.ThisHost.ID() {
			continue
		}

		// Create a new stream to the peer
		stream, err := peer.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("ProtocolSecondStep: Failed to create stream to peer %s: %v\n", peerInfo.ID, err)
			continue
		}
		defer stream.Close()

		// Send the deck
		_, err = stream.Write([]byte(fmt.Sprintf("CMDprotocolSS %s\n\\END\n", peer.Deck.GenerateDeckPayload())))
		if err != nil {
			log.Printf("ProtocolSecondStep: Failed to send deck to peer %s: %v\n", peerInfo.ID, err)
		} else {
			fmt.Printf("ProtocolSecondStep: Sent deck to peer %s\n", peerInfo.ID)
		}

		// Get the response (the new deck)
		responseBytes, err := io.ReadAll(stream)
		if err != nil {
			log.Printf("ProtocolSecondStep: Failed to read response from host %s: %v\n", peer.sessionHost, err)
		} else {
			peer.Deck.SetDeck(string(responseBytes)) // Setting the deck without changing it as no shuffling was done
			fmt.Printf("ProtocolSecondStep: Received response from peer %s\n", peerInfo.ID)
		}
	}
}

// Respond to the protocols second command - Decrypt global keys, encrypt with variation, then send back- when this is called a new deck should be set already
func (sp *ProtocolSecondStepCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	peer.DecryptAllWithGlobalKeys()
	peer.EncryptAllWithVariation()
	processedDeck := peer.Deck.GenerateDeckPayload()

	// Send the updated deck back to the sender
	_, err := sendingStream.Write([]byte(fmt.Sprintf("%s\\END\n", processedDeck)))
	if err != nil {
		log.Printf("StartProtocol Respond: Failed to send response: %v\n", err)
	}
}

//////////////////////////////////////////// INIT TABLE COMMAND /////////////////////////////////////////////////////

type InitTableCommand struct{}

func (it *InitTableCommand) Execute(peer *GokerPeer) {
	peer.peerListMutex.Lock()
	defer peer.peerListMutex.Unlock()

	for _, peerInfo := range peer.peerList {
		if peerInfo.ID == peer.ThisHost.ID() {
			continue
		}

		// Create a new stream to the peer
		stream, err := peer.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("InitTableCommand: Failed to create stream to peer %s: %v\n", peerInfo.ID, err)
			continue
		}
		defer stream.Close()

		// Send table rules and the start command
		_, err = stream.Write([]byte(fmt.Sprintf("CMDinittable %s\n\\END\n", peer.gameState.GetTableRules())))
		if err != nil {
			log.Printf("InitTableCommand: Failed to send command to peer %s: %v\n", peerInfo.ID, err)
		} else {
			fmt.Printf("InitTableCommand: Sent command to peer %s\n", peerInfo.ID)
		}
	}
}

func (it *InitTableCommand) Respond(peer *GokerPeer, sendingStream network.Stream) { // TODO: This will need to sign the table rules
	//channelmanager.FNET_StartRoundChan <- true
}

//////////////////////////////////////////// REQUEST HAND COMMAND /////////////////////////////////////////////////////

type RequestHandCommand struct{}

func (rh *RequestHandCommand) Execute(peer *GokerPeer) {
	peer.peerListMutex.Lock()
	defer peer.peerListMutex.Unlock()

	for _, peerInfo := range peer.peerList {
		if peerInfo.ID == peer.ThisHost.ID() {
			continue
		}

		stream, err := peer.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("RequestHandCommand: Failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		_, err = stream.Write([]byte("CMDrequesthand\n\\END\n"))
		if err != nil {
			log.Printf("RequestHandCommand: Failed to send command to peer%s: %v\n", peerInfo.ID, err)
		} else {
			fmt.Printf("RequestHandCommand: Sent command to peer%s\n", peerInfo.ID)
		}

		// Read the response - we do this as they won't be sending an actual *command* back, just some text
		responseBytes, err := io.ReadAll(stream)
		if err != nil {
			log.Printf("RequestHandCommand: Failed to read response from host %s: %v\n", peerInfo.ID, err)
		} else {
			keys := strings.Split(string(responseBytes), "\n")
			fmt.Print("KEYS SENT BY A PEER: ")
			fmt.Println(keys)
			// TODO: Decrypt my cards with this peers keys
			//peer.DecryptMyHand(keys[0], keys[1])
		}
	}
}

// Will do nothing
func (rh *RequestHandCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {}

//////////////////////////////////////////// ROUND COMMANDS /////////////////////////////////////////////////////

// TODO: Implement these.
type RaiseCommand struct{}
type CheckCommand struct{}
type CallCommand struct{}
type FoldCommand struct{}
