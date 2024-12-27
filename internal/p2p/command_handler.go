package p2p

import (
	"context"
	"fmt"
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
		log.Printf("PQRequestResponse: Failed to send PQ to peer %s: %v", sendingStream.Conn().RemotePeer(), err)
	} else {
		fmt.Printf("PQRequestResponse: Sent PQ to peer %s", sendingStream.Conn().RemotePeer())
	}
}

// / ### PROTOCOL ###
type StartProtocolCommand struct{}

// Send deck to every peer, allow them to shuffle and encrypt the deck
func (sp *StartProtocolCommand) Execute(peer *GokerPeer) {
	peer.peerListMutex.Lock()
	defer peer.peerListMutex.Unlock()
	fmt.Println(peer.gameInfo.GenerateDeckPayload())

	for peerID := range peer.peerList {
		if peerID == peer.thisHost.ID() {
			continue
		}

		// Create a new stream to the peer
		stream, err := peer.thisHost.NewStream(context.Background(), peerID, protocolID)
		if err != nil {
			log.Printf("PingCommand: Failed to create stream to peer %s: %v\n", peerID, err)
			continue
		}
		defer stream.Close()

		// Send the deck TODO: THIS IS BEING CAUGHT AS A PROBLEM ON THE SENDING PEER
		_, err = stream.Write([]byte(fmt.Sprintf("CMDstartprotocol %s\n\\END\n", peer.gameInfo.GenerateDeckPayload())))
		if err != nil {
			log.Printf("StartProtocol: Failed to send deck to peer %s: %v\n", peerID, err)
		} else {
			fmt.Printf("StartProtocol: Sent deck to peer %s\n", peerID)
		}

		// Get the response (the new deck)
		responseBytes, err := io.ReadAll(stream)
		if err != nil {
			log.Printf("StartProtocol: Failed to read response from host %s: %v\n", peer.sessionHost, err)
		} else {
			peer.gameInfo.SetDeck(string(responseBytes))
			fmt.Printf("StartProtocol: Received response from peer %s: %s\n", peerID, string(responseBytes))
		}

	}
	peer.gameInfo.DisplayDeck()
}

// Respond to a start protocol command - when this is called a new deck should be set already
func (sp *StartProtocolCommand) Respond(peer *GokerPeer, sendingStream network.Stream) {
	peer.gameInfo.DisplayDeck()
	// Encrypt the deck with your global keys, shuffle it, then create a new payload to send back
	peer.gameInfo.RoundDeck = peer.keyring.EncryptAllWithGlobalKeys(peer.gameInfo.RoundDeck)
	peer.gameInfo.ShuffleRoundDeck()
	processedDeck := peer.gameInfo.GenerateDeckPayload()

	// Send the updated deck back to the sender
	_, err := sendingStream.Write([]byte(fmt.Sprintf("%s\\END\n", processedDeck)))
	if err != nil {
		log.Printf("StartProtocol Respond: Failed to send response: %v\n", err)
	}
}
