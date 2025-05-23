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
	Command   string  `json:"command"`
	Payload   any     `json:"payload"`
	Signature string  `json:"signature"`
	Tag       *uint64 `json:"tag,omitempty"` // omitempty makes the tag field 'optional' (i.e. it won't even show up if there is nothing there)
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

func (p *GokerPeer) signCommand(nCmd *NetworkCommand) {
	signingData := nCmd.Command
	if nCmd.Payload != nil {
		payloadJSON, _ := json.Marshal(nCmd.Payload)
		signingData += string(payloadJSON)
	}

	if nCmd.Tag != nil { // Add tag if present
		signingData += fmt.Sprintf("%d", *nCmd.Tag)
	}

	signature, err := p.Keyring.SignMessage(signingData)
	if err != nil {
		log.Fatalf("signCommand: failed to sign request: %v", err)
	}
	// Attach signature inside the request
	nCmd.Signature = signature
}

func (p *GokerPeer) verifyCommand(from peer.ID, nCmd *NetworkCommand) {
	signingData := nCmd.Command
	if nCmd.Payload != nil {
		payloadJSON, _ := json.Marshal(nCmd.Payload)
		signingData += string(payloadJSON)
	}
	if nCmd.Tag != nil { // Add tag if present
		signingData += fmt.Sprintf("%d", *nCmd.Tag)
	}

	if !p.Keyring.VerifySignature(from, signingData, nCmd.Signature) {
		log.Println(nCmd.Command)
		log.Fatalf("verifyCommand: invalid signature in response from %s\n", from)
	}

	// Ensure game commands always have a tag
	gameCommands := []string{"Raise", "Check", "Call", "Fold"}
	for _, cmd := range gameCommands {
		if strings.EqualFold(nCmd.Command, cmd) {
			if nCmd.Tag == nil {
				log.Fatalf("verifyCommand: missing tag for game command: %s\n", nCmd.Command)
			}
			if *nCmd.Tag != p.tag {
				log.Fatalf("verifyCommand: invalid tag %d for game command: %s (expected %d)\n", *nCmd.Tag, nCmd.Command, p.tag)
			}
		}
	}
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

	// If it's PushTag, update p.tag first before verifying
	if nCmd.Command == "PushTag" && nCmd.Tag != nil {
		p.tag = *nCmd.Tag
	}

	if nCmd.Command != "GetPeers" && nCmd.Command != "PubKeyExchange" {
		p.verifyCommand(stream.Conn().RemotePeer(), &nCmd)
	}

	log.Println("Handling response to: " + nCmd.Command)

	// Process the command based on the message
	// These commands are in order for which they should be called
	switch nCmd.Command {
	case "GetPeers":
		p.RespondToCommand(&GetPeerListCommand{}, stream)
	case "PubKeyExchange":
		p.Keyring.SetPeerPublicKey(stream.Conn().RemotePeer(), nCmd.Payload.(string))
		p.RespondToCommand(&PubKeyExchangeCommand{}, stream)
	case "NicknameRequest":
		p.RespondToCommand(&NicknameRequestCommand{}, stream)
	case "InitTable":
		p.SetTurnOrderWithLobby()
		p.gameState.FreshStateFromPayload(nCmd.Payload.(string))
		channelmanager.TGUI_PlayerInfo <- p.gameState.GetPlayerInfo()
		p.RespondToCommand(&InitTableCommand{}, stream) // Respond with DONE
	case "SendPQ":
		channelmanager.TGUI_ShowLoadingChan <- struct{}{}
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
		p.SetHands()
		p.SetBoard()
		p.RespondToCommand(&BroadcastDeck{}, stream)
	case "PushTag":
		return
	case "CanRequestHand":
		p.ExecuteCommand(&RequestHandCommand{})
	case "RequestHand": // Someone is requesting the keys to their hand
		p.RespondToCommand(&RequestHandCommand{}, stream)
	case "MoveToTable":
		channelmanager.TGUI_StartRound <- struct{}{} // Tell GUI to move to the table UI
	case "Raise":
		p.gameState.PlayerRaise(stream.Conn().RemotePeer(), nCmd.Payload.(float64))
		p.RespondToCommand(&RaiseCommand{}, stream)
		p.gameState.NextTurn()
	case "Fold":
		p.DecryptRoundDeckWithPayload(nCmd.Payload.(string))
		p.gameState.PlayerFold(stream.Conn().RemotePeer())
		p.RespondToCommand(&FoldCommand{}, stream)
		p.gameState.NextTurn()
	case "Call":
		p.gameState.PlayerCall(stream.Conn().RemotePeer())
		p.RespondToCommand(&CallCommand{}, stream)
		p.gameState.NextTurn()
	case "Check":
		p.gameState.PlayerCheck(stream.Conn().RemotePeer())
		p.RespondToCommand(&CheckCommand{}, stream)
		p.gameState.NextTurn()
	case "RequestFlop":
		p.RespondToCommand(&RequestFlop{}, stream)
	case "RequestTurn":
		p.RespondToCommand(&RequestTurn{}, stream)
	case "RequestRiver":
		p.RespondToCommand(&RequestRiver{}, stream)
	case "RequestOthersHand":
		p.RespondToCommand(&RequestOthersHands{}, stream)
	case "CanRequestPuzzle":
		p.ExecuteCommand(&RequestPuzzleCommand{})
	case "PuzzleExchange":
		p.RespondToCommand(&RequestPuzzleCommand{}, stream)
	default:
		log.Printf("Unknown Command Recieved: %s\n", nCmd.Command)
	}
}

//////////////////////////////////////////// NEGOTIATION ///////////////////////////////////////////////////

// Sent to host to get the most up to date peer list
type GetPeerListCommand struct{}

func (gpl *GetPeerListCommand) Execute(p *GokerPeer) {
	stream, err := p.ThisHost.NewStream(context.Background(), p.sessionHost.ID, protocolID)
	if err != nil {
		log.Printf("GetPeerListCommand: failed to create stream to host %s: %v\n", p.sessionHost, err)
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

	p.setPeerListAndConnect(peerList)
	log.Println("GetPeerListCommand: received and set peerlist.")
}

func (gpl *GetPeerListCommand) Respond(p *GokerPeer, sendingStream network.Stream) {
	defer sendingStream.Close()

	response := NetworkCommand{
		Command: "GetPeers",
		Payload: p.getPeerList(),
	}

	if err := sendCommand(sendingStream, response); err != nil {
		log.Printf("GetPeerListCommand: failed to send peer list: %v", err)
	}
}

// Get signature data
type PubKeyExchangeCommand struct{}

func (pke *PubKeyExchangeCommand) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	key, err := p.Keyring.ExportPublicKey()
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	request := NetworkCommand{
		Command: "PubKeyExchange",
		Payload: key,
	}

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}
		if _, ok := p.Keyring.Otherskeys[peerInfo.ID]; ok { // If their pub key already exists, skip
			continue
		}

		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("PubKeyExchangeCommand: failed to create stream to peer %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		if err := sendCommand(stream, request); err != nil {
			log.Printf("PubKeyExchangeCommand: failed to send request: %v", err)
		}

		response, err := receiveResponse(stream)
		if err != nil {
			log.Fatalf("PubKeyExchangeCommand: failed to read response from peer %s: %v\n", peerInfo.ID, err)
		}

		keyPayload, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("PubKeyExchangeCommand: keys not a string from peer: %v\n", peerInfo.ID)
		}

		p.Keyring.SetPeerPublicKey(peerInfo.ID, keyPayload)
	}
}

func (pke *PubKeyExchangeCommand) Respond(p *GokerPeer, sendingStream network.Stream) {
	defer sendingStream.Close()

	key, err := p.Keyring.ExportPublicKey()
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	// Respond with the peer's public key
	response := NetworkCommand{
		Command: "PubKeyExchange",
		Payload: key,
	}

	if err := sendCommand(sendingStream, response); err != nil {
		log.Printf("RequestPublicKeyCommand: failed to send public key: %v", err)
	}
}

// Sent to everyone new joining to add to state
type NicknameRequestCommand struct{}

func (nr *NicknameRequestCommand) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	for _, peerInfo := range p.peerList {
		// If it's us
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}
		// if the player already exists, then obviously we don't need their nickname
		if p.gameState.PlayerExists(peerInfo.ID) {
			continue
		}

		// Create a new stream to the peer
		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("NicknameRequest: failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		// Create command
		command := NetworkCommand{
			Command: "NicknameRequest",
			Payload: nil,
		}
		p.signCommand(&command)

		// Send command through stream
		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("NicknameRequest: failed to send NicknameRequest command: %v", err)
		}

		// Receive and decode the response
		response, err := receiveResponse(stream)
		if err != nil {
			log.Fatalf("NicknameRequest: failed to read response from host %s: %v\n", peerInfo.ID, err)
		}
		p.verifyCommand(peerInfo.ID, &response)

		// Ensure the response payload is a string
		nickname, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("NicknameRequest: nickname not a string from peer: %v\n", peerInfo.ID)
		}

		peerNickname := strings.Split(nickname, "\n")
		log.Printf("NicknameRequest: Received response from peer: %s -- Nickname: %s\n", peerInfo.ID, peerNickname[0])
		p.gameState.AddPeerToState(peerInfo.ID, peerNickname[0]) // Finally add peer to gamestate
	}
}

func (nr *NicknameRequestCommand) Respond(p *GokerPeer, sendingStream network.Stream) {
	defer sendingStream.Close()

	response := NetworkCommand{
		Command: "NicknameRequest",
		Payload: p.gameState.GetNickname(p.ThisHost.ID()),
	}
	p.signCommand(&response)

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
			p.signCommand(&command)

			if err := sendCommand(stream, command); err != nil {
				log.Fatalf("InitTableCommand: failed to send getpeers command: %v", err)
			}

			response, err := receiveResponse(stream)
			if err != nil {
				log.Fatalf("InitTableCommand: failed to recieve response from peer: %v", err)
			}
			p.verifyCommand(peerID, &response)

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

func (it *InitTableCommand) Respond(p *GokerPeer, sendingStream network.Stream) {
	response := NetworkCommand{
		Command: "InitTable",
		Payload: "DONE",
	}
	p.signCommand(&response)

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("InitTableCommand: failed to send 'DONE': %v", err)
	}
}

////////////////////////////////////////// KEYRING //////////////////////////////////////////////////////

// Send P and Q to everyone for this rounds keyring
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
			p.signCommand(&command)

			if err := sendCommand(stream, command); err != nil {
				log.Fatalf("SendPQCommand: failed to send SendPQ command: %v", err)
			}

			response, err := receiveResponse(stream)
			if err != nil {
				log.Fatalf("SendPQCommand: failed to recieve response from peer: %v", err)
			}
			p.verifyCommand(peerID, &response)

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

func (pq *SendPQCommand) Respond(p *GokerPeer, sendingStream network.Stream) {
	response := NetworkCommand{
		Command: "SendPQ",
		Payload: "DONE",
	}
	p.signCommand(&response)

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

	// So I don't have to set the deck each turn
	command := NetworkCommand{
		Command: "ProtocolFS",
		Payload: p.Deck.GenerateDeckPayload(),
	}
	p.signCommand(&command)

	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()
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
		p.verifyCommand(peerInfo.ID, &response)

		newDeck, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("ProtocolFirstStep: invalid response format: expected string, got %T", response.Payload)
		}

		command.Payload = newDeck // So the next peer has an up to date deck
		p.signCommand(&command)
	}

	p.Deck.SetNewDeck(command.Payload.(string)) // Set the final deck for host
	log.Println("ProtocolFirstStepCommand: All peers have contributed, continueing...")
}

// Respond to a protocol's first command - Encrypt with global keys, shuffle, then send back - when this is called a new deck should be set already
func (sp *ProtocolFirstStepCommand) Respond(p *GokerPeer, sendingStream network.Stream) {
	// Encrypt the deck with your global keys, shuffle it, then create a new payload to send back
	p.EncryptAllWithGlobalKeys()
	p.Deck.ShuffleRoundDeck()

	response := NetworkCommand{
		Command: "ProtocolFirstStep",
		Payload: p.Deck.GenerateDeckPayload(),
	}
	p.signCommand(&response)

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
	p.signCommand(&command)

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
			p.verifyCommand(peerID, &response)

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
	p.signCommand(&response)

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
	p.signCommand(&command)

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
		p.verifyCommand(peerInfo.ID, &response)

		newDeck, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("ProtocolSecondStep: invalid response format: expected string, got %T", response.Payload)
		}

		command.Payload = newDeck
		p.signCommand(&command)
	}

	p.Deck.SetDeckInPlace(command.Payload.(string))
	p.SetHands() // Time to set my own hand
	p.SetBoard()
	log.Println("ProtocolSecondStepCommand: All peers have contributed, continueing...")
}

// Respond to the protocols second command - Decrypt global keys, encrypt with variation, then send back- when this is called a new deck should be set already
func (sp *ProtocolSecondStepCommand) Respond(p *GokerPeer, sendingStream network.Stream) {
	p.DecryptAllWithGlobalKeys()
	p.EncryptAllWithVariation()

	response := NetworkCommand{
		Command: "ProtocolSecondStep",
		Payload: p.Deck.GenerateDeckPayload(),
	}
	p.signCommand(&response)

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
	p.signCommand(&command)

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
			p.verifyCommand(peerID, &response)

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
	p.signCommand(&response)

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("BroadcastDeck: failed to send 'DONE' back to peer: %v", err)
	}
}

//////////////////////////////////////////// DEALING COMMAND /////////////////////////////////////////////////////

// To alert everyone they can request their hand
type CanRequestHand struct{}

func (c *CanRequestHand) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	command := NetworkCommand{
		Command: "CanRequestHand",
		Payload: nil,
	}
	p.signCommand(&command)

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

// Used if it's your turn to request your hand
type RequestHandCommand struct{}

func (rh *RequestHandCommand) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	var cardOneKeys []string
	var cardTwoKeys []string

	command := NetworkCommand{
		Command: "RequestHand",
		Payload: nil,
	}
	p.signCommand(&command)

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}

		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
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
		p.verifyCommand(peerInfo.ID, &response)

		keyPayload, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("RequestHand: invalid response format: expected string, got %T", response.Payload)
		}

		keys := strings.Split(keyPayload, "\n")
		cardOneKeys = append(cardOneKeys, keys[0])
		cardTwoKeys = append(cardTwoKeys, keys[1])
	}

	// Now that I have all the keys for my hand, decrypt the hand
	p.DecryptMyHand(cardOneKeys, cardTwoKeys)

	// Set the hand in the GUI
	cardOneName, exists := p.Deck.GetCardFromRefDeck(p.MyHand[0].CardValue) // Should be the hash
	cardTwoName, exists2 := p.Deck.GetCardFromRefDeck(p.MyHand[1].CardValue)

	if exists && exists2 {
		p.sendHandToGUI(cardOneName, cardTwoName)
		return
	}

	log.Fatalf("RequestHand: could not retrieve keys, aborting.")
}

func (rh *RequestHandCommand) Respond(p *GokerPeer, sendingStream network.Stream) {
	response := NetworkCommand{
		Command: "RequestHand",
		Payload: p.GetKeyPayloadForPlayersHand(sendingStream.Conn().RemotePeer()),
	}
	p.signCommand(&response)

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("RequestHand: failed to send keys back to peer: %v", err)
	}
}

//////////////////////////////////////////// ROUND COMMANDS /////////////////////////////////////////////////////

type PushTagCommand struct{}

func (pt *PushTagCommand) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	p.GenerateNewTag()

	command := NetworkCommand{
		Command: "PushTag",
		Payload: nil,
		Tag:     &p.tag,
	}
	p.signCommand(&command)

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}

		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("MoveToTable: Failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("MoveToTable: failed to send command to peer %s: %v", peerInfo.ID, err)
		}
	}
}

func (pt *PushTagCommand) Respond(p *GokerPeer, sendingStream network.Stream) {}

type MoveToTableCommand struct{}

func (mtt *MoveToTableCommand) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	command := NetworkCommand{
		Command: "MoveToTable",
		Payload: nil,
	}
	p.signCommand(&command)

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}

		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
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

func (mtt *MoveToTableCommand) Respond(p *GokerPeer, sendingStream network.Stream) {}

type RaiseCommand struct{}

func (r *RaiseCommand) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	command := NetworkCommand{
		Command: "Raise",
		Payload: p.gameState.MyBet,
		Tag:     &p.tag,
	}
	p.signCommand(&command)

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
		p.verifyCommand(peerInfo.ID, &response)

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
		Tag:     &p.tag,
	}
	p.signCommand(&response)

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
		Payload: p.Keyring.KeyringPayload,
		Tag:     &p.tag,
	}
	p.signCommand(&command)

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
		p.verifyCommand(peerInfo.ID, &response)

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
		Tag:     &p.tag,
	}
	p.signCommand(&response)

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
		Payload: nil,
		Tag:     &p.tag,
	}
	p.signCommand(&command)

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
		p.verifyCommand(peerInfo.ID, &response)

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
		Tag:     &p.tag,
	}
	p.signCommand(&response)

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
		Payload: nil,
		Tag:     &p.tag,
	}
	p.signCommand(&command)

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
		p.verifyCommand(peerInfo.ID, &response)

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
		Tag:     &p.tag,
	}
	p.signCommand(&response)

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("Check: failed to send 'APPROVED': %v", err)
	}

}

//////////////////////////////////////////// PHASE COMMANDS /////////////////////////////////////////////////////

type RequestFlop struct{}

func (rf *RequestFlop) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	var cardOneKeys []string
	var cardTwoKeys []string
	var cardThreeKeys []string

	command := NetworkCommand{
		Command: "RequestFlop",
		Payload: nil,
	}
	p.signCommand(&command)

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() || p.gameState.FoldedPlayers[peerInfo.ID] { // If it's us or the person has folded
			continue
		}

		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("RequestFlop: Failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("RequestFlop: failed to send command to peer %s: %v", peerInfo.ID, err)
		}

		response, err := receiveResponse(stream)
		if err != nil {
			log.Fatalf("RequestFlop: failed to recieve response from peer: %s", peerInfo.ID)
		}
		p.verifyCommand(peerInfo.ID, &response)

		keyPayload, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("RequestFlop: invalid response format: expected string, got %T", response.Payload)
		}

		keys := strings.Split(keyPayload, "\n")
		cardOneKeys = append(cardOneKeys, keys[0])
		cardTwoKeys = append(cardTwoKeys, keys[1])
		cardThreeKeys = append(cardThreeKeys, keys[2])
	}

	p.DecryptFlop(cardOneKeys, cardTwoKeys, cardThreeKeys)

	cardOneName, exists := p.Deck.GetCardFromRefDeck(p.Flop[0].CardValue)
	cardTwoName, exists2 := p.Deck.GetCardFromRefDeck(p.Flop[1].CardValue)
	cardThreeName, exists3 := p.Deck.GetCardFromRefDeck(p.Flop[2].CardValue)

	if exists && exists2 && exists3 {
		p.sendBoardToGUI(&cardOneName, &cardTwoName, &cardThreeName, nil, nil)
		return
	}

	log.Fatalf("RequestFlop: could not retrieve keys, aborting.")
}

func (rf *RequestFlop) Respond(p *GokerPeer, sendingStream network.Stream) {
	payload := p.GetKeyPayloadForFlop()
	response := NetworkCommand{
		Command: "RequestFlop",
		Payload: payload,
	}
	p.signCommand(&response)

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("RequestFlop: failed to send keys back to peer: %v", err)
	}
}

type RequestTurn struct{}

func (rt *RequestTurn) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	var turnKeys []string

	command := NetworkCommand{
		Command: "RequestTurn",
		Payload: nil,
	}
	p.signCommand(&command)

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() || p.gameState.FoldedPlayers[peerInfo.ID] {
			continue
		}

		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("RequestTurn: Failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("RequestTurn: failed to send command to peer %s: %v", peerInfo.ID, err)
		}

		response, err := receiveResponse(stream)
		if err != nil {
			log.Fatalf("RequestTurn: failed to recieve response from peer: %s", peerInfo.ID)
		}
		p.verifyCommand(peerInfo.ID, &response)

		keyPayload, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("RequestTurn: invalid response format: expected string, got %T", response.Payload)
		}

		turnKeys = append(turnKeys, keyPayload) // since it should only be one key at at ime
	}

	p.DecryptTurn(turnKeys)

	cardOneName, exists := p.Deck.GetCardFromRefDeck(p.Flop[0].CardValue)
	cardTwoName, exists1 := p.Deck.GetCardFromRefDeck(p.Flop[1].CardValue)
	cardThreeName, exists2 := p.Deck.GetCardFromRefDeck(p.Flop[2].CardValue)
	cardFourName, exists3 := p.Deck.GetCardFromRefDeck(p.Turn.CardValue)

	if exists && exists1 && exists2 && exists3 {
		p.sendBoardToGUI(&cardOneName, &cardTwoName, &cardThreeName, &cardFourName, nil)
		return
	}

	log.Fatalf("RequestTurn: could not retrieve keys, aborting.")
}

func (rt *RequestTurn) Respond(p *GokerPeer, sendingStream network.Stream) {
	payload := p.GetKeyPayloadForTurn()
	response := NetworkCommand{
		Command: "RequestTurn",
		Payload: payload,
	}
	p.signCommand(&response)

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("RequestTurn: failed to send keys back to peer: %v", err)
	}
}

type RequestRiver struct{}

func (rr *RequestRiver) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	var riverKeys []string

	command := NetworkCommand{
		Command: "RequestRiver",
		Payload: nil,
	}
	p.signCommand(&command)

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() || p.gameState.FoldedPlayers[peerInfo.ID] {
			continue
		}

		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("RequestRiver: Failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("RequestRiver: failed to send command to peer %s: %v", peerInfo.ID, err)
		}

		response, err := receiveResponse(stream)
		if err != nil {
			log.Fatalf("RequestRiver: failed to recieve response from peer: %s", peerInfo.ID)
		}
		p.verifyCommand(peerInfo.ID, &response)

		keyPayload, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("RequestRiver: invalid response format: expected string, got %T", response.Payload)
		}

		riverKeys = append(riverKeys, keyPayload)
	}

	p.DecryptRiver(riverKeys)

	cardOneName, exists := p.Deck.GetCardFromRefDeck(p.Flop[0].CardValue)
	cardTwoName, exists1 := p.Deck.GetCardFromRefDeck(p.Flop[1].CardValue)
	cardThreeName, exists2 := p.Deck.GetCardFromRefDeck(p.Flop[2].CardValue)
	cardFourName, exists3 := p.Deck.GetCardFromRefDeck(p.Turn.CardValue)
	cardFiveName, exists4 := p.Deck.GetCardFromRefDeck(p.River.CardValue)

	if exists && exists1 && exists2 && exists3 && exists4 {
		p.sendBoardToGUI(&cardOneName, &cardTwoName, &cardThreeName, &cardFourName, &cardFiveName)
		return
	}

	log.Fatalf("RequestRiver: could not retrieve keys, aborting.")
}

func (rr *RequestRiver) Respond(p *GokerPeer, sendingStream network.Stream) {
	payload := p.GetKeyPayloadForRiver()
	response := NetworkCommand{
		Command: "RequestRiver",
		Payload: payload,
	}
	p.signCommand(&response)

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("RequestRiver: failed to send keys back to peer: %v", err)
	}
}

//////////////////////////////////////////// END ROUND COMMANDS /////////////////////////////////////////////////////

type RequestOthersHands struct{}

func (r *RequestOthersHands) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	command := NetworkCommand{
		Command: "RequestOthersHand",
		Payload: nil,
	}
	p.signCommand(&command)

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() || p.gameState.FoldedPlayers[peerInfo.ID] {
			continue
		}

		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
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
		p.verifyCommand(peerInfo.ID, &response)

		keyPayload, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("RequestHand: invalid response format: expected string, got %T", response.Payload)
		}

		p.DecryptOthersHand(peerInfo.ID, strings.Split(keyPayload, "\n"))
	}
}

func (rh *RequestOthersHands) Respond(p *GokerPeer, sendingStream network.Stream) {
	payload := p.GetKeyPayloadForMyHand()
	response := NetworkCommand{
		Command: "RequestOthersHand",
		Payload: payload,
	}
	p.signCommand(&response)

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("RequestOthersHand: failed to send keys back to peer: %v", err)
	}
}

//////////////////////////////////////////// TLP COMMANDS /////////////////////////////////////////////////////

type CanRequestPuzzle struct{}

func (c *CanRequestPuzzle) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	command := NetworkCommand{
		Command: "CanRequestPuzzle",
		Payload: nil,
	}
	p.signCommand(&command)

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() {
			continue
		}
		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)

		if err != nil {
			log.Printf("CanRequstPuzzle: failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()
		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("CanRequestPuzzle: failed to send command to peer %s: %v", peerInfo.ID, err)
		}
	}
}

func (c *CanRequestPuzzle) Respond(p *GokerPeer, sendingStream network.Stream) {}

type RequestPuzzleCommand struct{}

func (tlp *RequestPuzzleCommand) Execute(p *GokerPeer) {
	p.peerListMutex.Lock()
	defer p.peerListMutex.Unlock()

	command := NetworkCommand{
		Command: "PuzzleExchange",
		Payload: nil,
	}
	p.signCommand(&command)

	for _, peerInfo := range p.peerList {
		if peerInfo.ID == p.ThisHost.ID() || p.gameState.FoldedPlayers[peerInfo.ID] {
			continue
		}

		stream, err := p.ThisHost.NewStream(context.Background(), peerInfo.ID, protocolID)
		if err != nil {
			log.Printf("PuzzleExchange: Failed to create stream to host %s: %v\n", peerInfo.ID, err)
			return
		}
		defer stream.Close()

		if err := sendCommand(stream, command); err != nil {
			log.Fatalf("PuzzleExchange: failed to send command to peer %s: %v", peerInfo.ID, err)
		}

		response, err := receiveResponse(stream)
		if err != nil {
			log.Fatalf("PuzzleExchange: failed to recieve response from peer: %s", peerInfo.ID)
		}
		p.verifyCommand(peerInfo.ID, &response)

		puzzlePayload, ok := response.Payload.(string)
		if !ok {
			log.Fatalf("PuzzleExchange: invalid response format: expected string, got %T", response.Payload)
		}

		go p.BreakTimeLockedPuzzle(peerInfo.ID, []byte(puzzlePayload))
	}
}

func (tlp *RequestPuzzleCommand) Respond(p *GokerPeer, sendingStream network.Stream) {
	numOfPlayers := p.gameState.GetNumberOfPlayers()
	numOfPhases := 4                                                            // Accounts for Preflop, Flop, Turn, and River
	p.Keyring.GenerateTimeLockedPuzzle(int64(15*numOfPlayers*numOfPhases + 30)) // + 30 to account for threshold

	payload, err := json.Marshal(p.Keyring.TLP)
	if err != nil {
		log.Fatalf("failed to serialize time-locked puzzle")
	}

	response := NetworkCommand{
		Command: "PuzzleExchange",
		Payload: string(payload),
	}
	p.signCommand(&response)

	if err := sendCommand(sendingStream, response); err != nil {
		log.Fatalf("RequestOthersHand: failed to send keys back to peer: %v", err)
	}
}
