package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"goker/internal/channelmanager"
	"log"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Uses Command Design Pattern
type PeerCommand interface {
	Execute(peer *GokerPeer)
	Respond(peer *GokerPeer, sendingStream network.Stream)
}

// Execute a command
func (p *GokerPeer) ExecuteCommand(pCmd PeerCommand) {
	pCmd.Execute(p)
}

// Respond to a command
func (p *GokerPeer) RespondToCommand(pCmd PeerCommand, stream network.Stream) {
	pCmd.Respond(p, stream)
}

// Structure for messages to be sent over the network
type NetworkCommand struct {
	Command string `json:"command"`
	Payload any    `json:"payload"`
}

// Send a network command to a specific stream
func sendCommand(stream network.Stream, nCmd NetworkCommand) error {
	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(nCmd); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}
	return nil
}

// Recieve a network command from a specifc stream
func receiveResponse(stream network.Stream) (NetworkCommand, error) {
	var cmd NetworkCommand
	decoder := json.NewDecoder(stream)

	if err := decoder.Decode(&cmd); err != nil {
		return NetworkCommand{}, fmt.Errorf("recieveResponse: failed to decode command: %w", err)
	}

	return cmd, nil
}

// Handle incoming streams (should be commands only)
func (p *GokerPeer) handleStream(stream network.Stream) {
	defer stream.Close()

	// Create a JSON decoder for reading the incoming command
	decoder := json.NewDecoder(stream)
	var nCmd NetworkCommand

	// Decode the command
	if err := decoder.Decode(&nCmd); err != nil {
		log.Printf("Failed to decode incoming network command: %v", err)
		return
	}

	// Process the command based on the message
	// These commands are in order for which they should be called
	switch nCmd.Command {
	case "GetPeers":
		p.RespondToCommand(&GetPeerListCommand{}, stream)
	case "NicknameRequest":
		p.RespondToCommand(&NicknameRequestCommand{}, stream)
	case "InitTable":
		p.SetTurnOrderWithLobby()
		p.gameState.FreshStateFromPayload(nCmd.Payload.(string))
		channelmanager.TGUI_PlayerInfo <- p.gameState.GetPlayerInfo()
		p.RespondToCommand(&InitTableCommand{}, stream) // Respond with DONE
	case "SendPQ":
		pq := strings.Split(string(nCmd.Payload.(string)), "\n")
		p.Keyring.SetPQ(pq[0], pq[1])
		p.Keyring.GenerateKeys()
		p.RespondToCommand(&SendPQCommand{}, stream) // Respond with DONE
	case "ProtocolFS": // First step of Protocol
		p.Deck.SetNewDeck(nCmd.Payload.(string))
		p.RespondToCommand(&ProtocolFirstStepCommand{}, stream)
	case "BroadcastNewDeck": // First shuffled deck (will be shuffled so needs to be set as a new deck)
		p.Deck.SetNewDeck(nCmd.Payload.(string))
		p.RespondToCommand(&BroadcastNewDeck{}, stream)
	case "ProtocolSS": // Second step of Protocol
		p.Deck.SetDeckInPlace(nCmd.Payload.(string))
		p.RespondToCommand(&ProtocolSecondStepCommand{}, stream)
	case "BroadcastDeck": // Final shuffled deck
		p.Deck.SetDeckInPlace(nCmd.Payload.(string))
		p.SetHands() // Make sure at the end of the second step of the protocol to set everyones hands for that round
		p.RespondToCommand(&BroadcastDeck{}, stream)
	case "CanRequestHand": // Once everythings ready this will be recieved from the host, so everyone can reqeust their cards
		p.ExecuteCommand(&RequestHandCommand{})
	case "RequestHand": // Someone is requesting the keys to their hand
		p.RespondToCommand(&RequestHandCommand{}, stream)
	case "MoveToTable":
		channelmanager.TGUI_StartRound <- struct{}{} // Tell GUI to move to the table UI
	case "Raise":
		p.gameState.PlayerRaise(stream.Conn().RemotePeer(), nCmd.Payload.(float64))
		p.gameState.NextTurn()
		p.RespondToCommand(&RaiseCommand{}, stream)
	case "Fold":
		p.gameState.PlayerFold(stream.Conn().RemotePeer())
		p.gameState.NextTurn()
		p.RespondToCommand(&FoldCommand{}, stream)
	case "Call":
		p.gameState.PlayerCall(stream.Conn().RemotePeer())
		p.gameState.NextTurn()
		p.RespondToCommand(&CallCommand{}, stream)
	case "Check":
		p.gameState.PlayerCheck(stream.Conn().RemotePeer())
		p.gameState.NextTurn()
		p.RespondToCommand(&CheckCommand{}, stream)
	default:
		log.Printf("Unknown Command Recieved: %s\n", nCmd.Command)
	}
}

//////////////////////////////////////////// NEGOTIATION ///////////////////////////////////////////////////

// Sent to host to get the most up to date peer list
type GetPeerListCommand struct{}

func (gpl *GetPeerListCommand) Execute(peer *GokerPeer) {
	stream, err := peer.ThisHost.NewStream(context.Background(), peer.sessionHost.ID, protocolID)
	if err != nil {
		log.Printf("GetPeerListCommand: failed to create stream to host %s: %v\n", peer.sessionHost, err)
		return
	}
	defer stream.Close()

	// Create and send the command
	request := NetworkCommand{
		Command: "GetPeers",
		Payload: nil, // No payload needed for this request
	}

	if err := sendCommand(stream, request); err != nil {
		log.Fatalf("GetPeerListCommand: failed to send getpeers command: %v", err)
	}

	// Receive and decode the response
	response, err := receiveResponse(stream)
	if err != nil {
		log.Fatalf("GetPeerListCommand: failed to receive peer list: %v", err)
	}

	// Ensure the response payload is a string
	peerList, ok := response.Payload.(string)
	if !ok {
		log.Fatalf("GetPeerListCommand: invalid response format: expected string, got %T", response.Payload)
	}

	peer.setPeerListAndConnect(peerList)
	log.Println("GetPeerListCommand: received and set peerlist.")
}

func (gpl *GetPeerListCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	defer sendingStream.Close()

	response := NetworkCommand{
		Command: "GetPeers",
		Payload: peer.getPeerList(),
	}

	if err := sendCommand(sendingStream, response); err != nil {
		log.Printf("GetPeerListCommand: failed to send peer list: %v", err)
	}
}

// Sent to everyone new joining to add to state
type NicknameRequestCommand struct{}

func (nr *NicknameRequestCommand) Execute(peer *GokerPeer) {
	peer.peerListMutex.Lock()
	defer peer.peerListMutex.Unlock()

	for _, peerInfo := range peer.peerList {
		// If it's us
		if peerInfo.ID == peer.ThisHost.ID() {
			continue
		}
		// if the player already exists, then obviously we don't need their nickname
		if peer.gameState.PlayerExists(peerInfo.ID) {
			continue
		}

		// Create a new stream to the peer
		stream, err := peer.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("NicknameRequest: failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		// Create command
		request := NetworkCommand{
			Command: "NicknameRequest",
			Payload: nil, // No payload needed for this request
		}

		// Send command through stream
		if err := sendCommand(stream, request); err != nil {
			log.Fatalf("NicknameRequest: failed to send NicknameRequest command: %v", err)
		}

		// Receive and decode the response
		response, err := receiveResponse(stream)
		if err != nil {
			log.Fatalf("NicknameRequest: failed to read response from host %s: %v\n", peerInfo.ID, err)
		}

		// Ensure the response payload is a string
		nickname, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("NicknameRequest: nickname not a string from peer: %v\n", peerInfo.ID)
		}

		peerNickname := strings.Split(nickname, "\n")
		log.Printf("NicknameRequest: Received response from peer: %s -- Nickname: %s\n", peerInfo.ID, peerNickname[0])
		peer.gameState.AddPeerToState(peerInfo.ID, peerNickname[0]) // Finally add peer to gamestate
	}
}

func (nr *NicknameRequestCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	defer sendingStream.Close()

	response := NetworkCommand{
		Command: "NicknameRequest",
		Payload: peer.gameState.GetNickname(peer.ThisHost.ID()),
	}

	if err := sendCommand(sendingStream, response); err != nil {
		log.Printf("NicknameRequestCommand: failed to send nickname: %v", err)
	}
}

//////////////////////////////////////////// INIT TABLE COMMAND /////////////////////////////////////////////////////

// Sent to everyone to set the table rules for the game
type InitTableCommand struct{}

func (it *InitTableCommand) Execute(p *GokerPeer) {
	var wg sync.WaitGroup

	p.peerListMutex.Lock()
	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}

		wg.Add(1)

		go func(peerID peer.ID) {
			defer wg.Done()
			// Create a new stream to the peer
			stream, err := p.ThisHost.NewStream(context.Background(), peerID, protocolID)
			if err != nil {
				log.Printf("InitTableCommand: Failed to create stream to peer %s: %v\n", peerID, err)
			}
			defer stream.Close()

			command := NetworkCommand{
				Command: "InitTable",
				Payload: p.gameState.GetTableRules(),
			}

			if err := sendCommand(stream, command); err != nil {
				log.Fatalf("InitTableCommand: failed to send getpeers command: %v", err)
			}

			response, err := receiveResponse(stream)
			if err != nil {
				log.Fatalf("InitTableCommand: failed to recieve response from peer: %v", err)
			}

			doneResponse, ok := response.Payload.(string)
			if !ok {
				log.Fatalf("InitTableCommand: invalid response format: expected string, got %T", response.Payload)
			}

			if doneResponse != "DONE" {
				log.Fatalf("InitTableCommand: response from peer was not 'DONE', got %s", doneResponse)
			}
		}(peerInfo.ID)
	}
	p.peerListMutex.Unlock()

	wg.Wait()
	log.Println("InitTableCommand: All available peers responded, proceeding...")
}

func (it *InitTableCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	response := NetworkCommand{
		Command: "InitTable",
		Payload: "DONE",
	}

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("InitTableCommand: failed to send 'DONE': %v", err)
	}
}

////////////////////////////////////////// KEYRING //////////////////////////////////////////////////////

// Send P and Q to everyone for this rounds keyring
// TODO: Find a better way to do this?
type SendPQCommand struct{}

func (pq *SendPQCommand) Execute(p *GokerPeer) {
	var wg sync.WaitGroup

	p.peerListMutex.Lock()
	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}

		wg.Add(1)

		go func(peerID peer.ID) {
			defer wg.Done()
			// Create a new stream to the peer
			stream, err := p.ThisHost.NewStream(context.Background(), peerID, protocolID)
			if err != nil {
				log.Printf("SendPQCommand: Failed to create stream to peer %s: %v\n", peerID, err)
			}
			defer stream.Close()

			command := NetworkCommand{
				Command: "SendPQ",
				Payload: p.Keyring.GetPQString(),
			}

			if err := sendCommand(stream, command); err != nil {
				log.Fatalf("SendPQCommand: failed to send SendPQ command: %v", err)
			}

			response, err := receiveResponse(stream)
			if err != nil {
				log.Fatalf("SendPQCommand: failed to recieve response from peer: %v", err)
			}

			doneResponse, ok := response.Payload.(string)
			if !ok {
				log.Fatalf("SendPQCommand: invalid response format: expected string, got %T", response.Payload)
			}

			if doneResponse != "DONE" {
				log.Fatalf("SendPQCommand: response from peer was not 'DONE', got %s", doneResponse)
			}
		}(peerInfo.ID)
	}
	p.peerListMutex.Unlock()

	wg.Wait()
	log.Println("SendPQCommand: All available peers responded, proceeding...")
}

func (pq *SendPQCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	response := NetworkCommand{
		Command: "SendPQ",
		Payload: "DONE",
	}

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("SendPQCommand: failed to send 'DONE': %v", err)
	}
}

///////////////////////////////////////////// PROTOCOL FIRST STEP ///////////////////////////////////////////////////

type ProtocolFirstStepCommand struct{}

// Send deck to every peer, allow them to shuffle and encrypt the deck
func (sp *ProtocolFirstStepCommand) Execute(p *GokerPeer) {
	p.EncryptAllWithGlobalKeys()
	p.Deck.ShuffleRoundDeck()

	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	// So I don't have to set the deck each turn
	command := NetworkCommand{
		Command: "ProtocolFS",
		Payload: p.Deck.GenerateDeckPayload(),
	}

	// Get each peer to shuffle and encrypt deck
	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}

		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("ProtocolFirstStep: Failed to create stream to peer %s: %v\n", peerInfo.ID, err)
			continue
		}
		defer stream.Close()

		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("ProtocolFirstStepCommand: failed to send deck: %v", err)
		}

		response, err := receiveResponse(stream)
		if err != nil {
			log.Fatalf("ProtocolFirstStepCommand: failed to recieve deck from peer: %v", err)
		}

		newDeck, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("ProtocolFirstStep: invalid response format: expected string, got %T", response.Payload)
		}

		command.Payload = newDeck // So the next peer has an up to date deck
	}

	p.Deck.SetNewDeck(command.Payload.(string)) // Set the final deck for host
	log.Println("ProtocolFirstStepCommand: All peers have contributed, continueing...")
}

// Respond to a protocol's first command - Encrypt with global keys, shuffle, then send back - when this is called a new deck should be set already
func (sp *ProtocolFirstStepCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	// Encrypt the deck with your global keys, shuffle it, then create a new payload to send back
	peer.EncryptAllWithGlobalKeys()
	peer.Deck.ShuffleRoundDeck()

	response := NetworkCommand{
		Command: "ProtocolFirstStep",
		Payload: peer.Deck.GenerateDeckPayload(),
	}

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("ProtocolFirstStep: failed to send 'DONE' back to peer: %v", err)
	}
}

// Send the new deck to everyone - Everyone will need to set the deck as if it was new
type BroadcastNewDeck struct{}

func (b *BroadcastNewDeck) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock() // Since this is called RIGHT after the first round of the protocol is done, it can unlock the peerlist

	var wg sync.WaitGroup

	command := NetworkCommand{
		Command: "BroadcastNewDeck",
		Payload: p.Deck.GenerateDeckPayload(),
	}

	// After all peers have processed, broadcast the final deck to everyone - This is where they will validate signatures?
	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}

		wg.Add(1)

		go func(peerID peer.ID) {
			defer wg.Done()

			stream, err := p.ThisHost.NewStream(context.Background(), peerID, protocolID)
			if err != nil {
				log.Fatalf("BroadcastNewDeck: failed to setup stream to peer %s: %v\n", peerID, err)
			}
			defer stream.Close()

			if err := sendCommand(stream, command); err != nil {
				log.Fatalf("BroadcastNewDeck: failed to send deck: %v", err)
			}

			response, err := receiveResponse(stream)
			if err != nil {
				log.Fatalf("BroadcastNewDeck: failed to recieve response from peer: %v", err)
			}

			doneResponse, ok := response.Payload.(string)
			if !ok {
				log.Fatalf("BroadcastNewDeck: invalid response format: expected string, got %T", response.Payload)
			}

			if doneResponse != "DONE" {
				log.Fatalf("BroadcastNewDeck: response from peer was not 'DONE', got %s", doneResponse)
			}

		}(peerInfo.ID)
	}

	wg.Wait()
	log.Println("BroadcastNewDeck: All peers have recieved deck, continueing...")
}

func (b *BroadcastNewDeck) Respond(p *GokerPeer, sendingStream network.Stream) {
	response := NetworkCommand{
		Command: "BroadcastNewDeck",
		Payload: "DONE",
	}

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("BroadcastNewDeck: failed to send 'DONE' back to peer: %v", err)
	}
}

///////////////////////////////////////////// PROTOCOL SECOND STEP ///////////////////////////////////////////////////

// Send deck to every peer, have them decrypt their global keys then encrypt the deck with variations and send back
type ProtocolSecondStepCommand struct{}

// For getting others to encrypt with their variation keys
func (sp *ProtocolSecondStepCommand) Execute(p *GokerPeer) {
	p.peerListMutex.Lock() // Same thing as first step, the broadcast will unlock the mutex
	defer p.peerListMutex.Unlock()

	// First, the host of the game (the one initialing the protocols steps) will encrypt with variations
	p.DecryptAllWithGlobalKeys() // Decrypt global keys
	p.EncryptAllWithVariation()  // Add encryption to every card

	command := NetworkCommand{
		Command: "ProtocolSS",
		Payload: p.Deck.GenerateDeckPayload(),
	}

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}

		// Create a new stream to the peer
		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("ProtocolSecondStep: Failed to create stream to peer %s: %v\n", peerInfo.ID, err)
			continue
		}
		defer stream.Close()

		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("ProtocolSecondStep: failed to send deck to peer")
		}

		response, err := receiveResponse(stream)
		if err != nil {
			log.Fatalf("ProtocolSecondStep: failed to receive a response from peer %s: %v", peerInfo.ID, err)
		}

		newDeck, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("ProtocolSecondStep: invalid response format: expected string, got %T", response.Payload)
		}

		command.Payload = newDeck
	}

	p.Deck.SetDeckInPlace(command.Payload.(string))
	p.SetHands() // Now that the new deck is in place, we can set hands
	log.Println("ProtocolSecondStepCommand: All peers have contributed, continueing...")
}

// Respond to the protocols second command - Decrypt global keys, encrypt with variation, then send back- when this is called a new deck should be set already
func (sp *ProtocolSecondStepCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	peer.DecryptAllWithGlobalKeys()
	peer.EncryptAllWithVariation()

	response := NetworkCommand{
		Command: "ProtocolSecondStep",
		Payload: peer.Deck.GenerateDeckPayload(),
	}

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("ProtocolSecondStep: failed to send deck back to peer: %v", err)
	}
}

// Send the unchanged (no shuffling) deck to everyone
type BroadcastDeck struct{}

func (b *BroadcastDeck) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	var wg sync.WaitGroup

	command := NetworkCommand{
		Command: "BroadcastDeck",
		Payload: p.Deck.GenerateDeckPayload(),
	}

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}

		wg.Add(1)

		go func(peerID peer.ID) {
			defer wg.Done()

			stream, err := p.ThisHost.NewStream(context.Background(), peerID, protocolID)
			if err != nil {
				log.Fatalf("BroadcastDeck: failed to setup stream to peer %s: %v\n", peerID, err)
			}
			defer stream.Close()

			if err := sendCommand(stream, command); err != nil {
				log.Fatalf("BroadcastDeck: failed to send deck: %v", err)
			}

			response, err := receiveResponse(stream)
			if err != nil {
				log.Fatalf("BroadcastDeck: failed to recieve response from peer: %v", err)
			}

			doneResponse, ok := response.Payload.(string)
			if !ok {
				log.Fatalf("BroadcastDeck: invalid response format: expected string, got %T", response.Payload)
			}

			if doneResponse != "DONE" {
				log.Fatalf("BroadcastDeck: response from peer was not 'DONE', got %s", doneResponse)
			}

		}(peerInfo.ID)
	}

	wg.Wait()
	log.Println("BroadcastDeck: All peers have recieved final deck, continueing...")
}

func (b *BroadcastDeck) Respond(p *GokerPeer, sendingStream network.Stream) {
	response := NetworkCommand{
		Command: "BroadcastDeck",
		Payload: "DONE",
	}

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("BroadcastDeck: failed to send 'DONE' back to peer: %v", err)
	}
}

//////////////////////////////////////////// REQUEST HAND COMMAND /////////////////////////////////////////////////////

// Updates everyone that they can request their hand
type CanRequestHand struct{}

func (c *CanRequestHand) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	command := NetworkCommand{
		Command: "CanRequestHand",
	}

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}

		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("CanRequestHand: failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("CanRequestHand: failed to send command to peer %s: %v", peerInfo.ID, err)
		}
	}
}

func (c *CanRequestHand) Respond(p *GokerPeer, sendingStream network.Stream) {}

type RequestHandCommand struct{}

func (rh *RequestHandCommand) Execute(peer *GokerPeer) {
	peer.peerListMutex.Lock()
	defer peer.peerListMutex.Unlock()

	var cardOneKeys []string
	var cardTwoKeys []string

	command := NetworkCommand{
		Command: "RequestHand",
	}

	for _, peerInfo := range peer.peerList {
		if peerInfo.ID == peer.ThisHost.ID() {
			continue
		}

		stream, err := peer.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("RequestHand: Failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("RequestHand: failed to send command to peer %s: %v", peerInfo.ID, err)
		}

		response, err := receiveResponse(stream)
		if err != nil {
			log.Fatalf("RequestHand: failed to recieve response from peer: %s", peerInfo.ID)
		}

		keyPayload, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("RequestHand: invalid response format: expected string, got %T", response.Payload)
		}

		keys := strings.Split(keyPayload, "\n")
		cardOneKeys = append(cardOneKeys, keys[0])
		cardTwoKeys = append(cardTwoKeys, keys[1])
	}

	fmt.Println("RECIEVED KEYS")
	fmt.Println(cardOneKeys)
	fmt.Println(cardTwoKeys)

	// Now that I have all the keys for my hand, decrypt the hand
	peer.DecryptMyHand(cardOneKeys, cardTwoKeys)

	// Set the hand in the GUI
	cardOneName, exists := peer.Deck.GetCardFromRefDeck(peer.MyHand.Hand[0].CardValue) // Should be the hash
	if !exists {
		log.Fatalf("RequestHand: couldn't find card one: %s", peer.MyHand.Hand[0].CardValue.String())
	}
	cardTwoName, exists := peer.Deck.GetCardFromRefDeck(peer.MyHand.Hand[1].CardValue)
	if !exists {
		log.Fatalf("RequestHand: couldn't find card two: %s", peer.MyHand.Hand[1].CardValue.String())
	}
	peer.sendHandToGUI(cardOneName, cardTwoName)
}

func (rh *RequestHandCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	payload := peer.GetKeyPayloadForPlayersHand(sendingStream.Conn().RemotePeer())
	fmt.Println("SENDING KEYS: " + payload)
	response := NetworkCommand{
		Command: "RequestHand",
		Payload: payload,
	}

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("RequestHand: failed to send keys back to peer: %v", err)
	}
}

//////////////////////////////////////////// ROUND COMMANDS /////////////////////////////////////////////////////

type MoveToTableCommand struct{}

func (mtt *MoveToTableCommand) Execute(peer *GokerPeer) {
	peer.peerListMutex.Lock()
	defer peer.peerListMutex.Unlock()

	command := NetworkCommand{
		Command: "MoveToTable",
	}

	for _, peerInfo := range peer.peerList {
		if peerInfo.ID == peer.ThisHost.ID() {
			continue
		}

		stream, err := peer.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("MoveToTable: Failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("MoveToTable: failed to send command to peer %s: %v", peerInfo.ID, err)
		}
	}

	channelmanager.TGUI_StartRound <- struct{}{} // Tell GUI to move to the table UI
}

func (mtt *MoveToTableCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {}

// TODO: Implement these.
type RaiseCommand struct{}

func (r *RaiseCommand) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	command := NetworkCommand{
		Command: "Raise",
		Payload: p.gameState.MylastBet,
	}

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}

		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("Raise: failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("Raise: failed to send command to peer %s: %v", peerInfo.ID, err)
		}

		response, err := receiveResponse(stream)
		if err != nil {
			log.Fatalf("Raise: failed to recieve a response from peer: %s: %v", peerInfo.ID, err)
		}

		approved, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("Raise: invalid response format: expected string, got %T", response.Payload)
		}

		if approved != "APPROVED" {
			log.Fatalf("Raise: Raise was not APPROVED, got %s", approved)
		}

	}
}

func (r *RaiseCommand) Respond(p *GokerPeer, sendingStream network.Stream) {
	response := NetworkCommand{
		Command: "Raise",
		Payload: "APPROVED",
	}

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("Raise: failed to send 'APPROVED': %v", err)
	}
}

type FoldCommand struct{}

func (f *FoldCommand) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	command := NetworkCommand{
		Command: "Fold",
	}

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}

		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("Fold: failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("Fold: failed to send command to peer %s: %v", peerInfo.ID, err)
		}

		response, err := receiveResponse(stream)
		if err != nil {
			log.Fatalf("Fold: failed to recieve a response from peer: %s: %v", peerInfo.ID, err)
		}

		approved, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("Fold: invalid response format: expected string, got %T", response.Payload)
		}

		if approved != "APPROVED" {
			log.Fatalf("Fold: Fold was not APPROVED, got %s", approved)
		}
	}
}

func (f *FoldCommand) Respond(p *GokerPeer, sendingStream network.Stream) {
	response := NetworkCommand{
		Command: "Fold",
		Payload: "APPROVED",
	}

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("Fold: failed to send 'APPROVED': %v", err)
	}
}

type CallCommand struct{}

func (c *CallCommand) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	command := NetworkCommand{
		Command: "Call",
	}

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}

		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("Call: failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("Call: failed to send command to peer %s: %v", peerInfo.ID, err)
		}

		response, err := receiveResponse(stream)
		if err != nil {
			log.Fatalf("Call: failed to recieve a response from peer: %s: %v", peerInfo.ID, err)
		}

		approved, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("Call: invalid response format: expected string, got %T", response.Payload)
		}

		if approved != "APPROVED" {
			log.Fatalf("Call: Call was not APPROVED, got %s", approved)
		}
	}
}

func (c *CallCommand) Respond(p *GokerPeer, sendingStream network.Stream) {
	response := NetworkCommand{
		Command: "Call",
		Payload: "APPROVED",
	}

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("Call: failed to send 'APPROVED': %v", err)
	}
}

type CheckCommand struct{}

func (c *CheckCommand) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	command := NetworkCommand{
		Command: "Check",
	}

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}

		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("Check: failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("Check: failed to send command to peer %s: %v", peerInfo.ID, err)
		}

		response, err := receiveResponse(stream)
		if err != nil {
			log.Fatalf("Check: failed to recieve a response from peer: %s: %v", peerInfo.ID, err)
		}

		approved, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("Check: invalid response format: expected string, got %T", response.Payload)
		}

		if approved != "APPROVED" {
			log.Fatalf("Check: Check was not APPROVED, got %s", approved)
		}
	}
}

func (c *CheckCommand) Respond(p *GokerPeer, sendingStream network.Stream) {
	response := NetworkCommand{
		Command: "Check",
		Payload: "APPROVED",
	}

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("Check: failed to send 'APPROVED': %v", err)
	}

}
