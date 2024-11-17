package p2p

import (
	"bufio"
	"context"
	"fmt"
	"goker/internal/sra"
	"io"
	"log"
	"strings"
	"math/big"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/network"
)

type Command interface {
	Execute(server *BootstrapServer)
	Respond(server *BootstrapServer, sendingStream network.Stream)
}

// Execute a command
func (s *BootstrapServer) ExecuteCommand(command Command) {
	command.Execute(s)
}

// Respond to a command
func (s *BootstrapServer) RespondToCommand(command Command, stream network.Stream) {
	command.Respond(s, stream)
}


// PingCommand struct inheriting from the Command Interface - For sending pings
type PingCommand struct{}
// Execute sends a ping command to all peers in the peer list
func (p *PingCommand) Execute(server *BootstrapServer) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	for peerID := range server.peers {
		if peerID == server.host.ID() {
			continue // Skip self
		}

		// Create a new stream to the peer
		stream, err := server.host.NewStream(context.Background(), peerID, protocolID)
		if err != nil {
			log.Printf("PingCommand: Failed to create stream to peer %s: %v\n", peerID, err)
			continue
		}
		defer stream.Close()

		// Send the "ping"
		_, err = stream.Write([]byte("CMDping\n"))
		if err != nil {
			log.Printf("PingCommand: Failed to send ping to peer %s: %v\n", peerID, err)
		} else {
			fmt.Printf("PingCommand: Sent ping to peer %s\n", peerID)
		}

		// Read the response - we do this as they won't be sending an actual *command* back, just some text
		reader := bufio.NewReader(stream)
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("PingCommand: Failed to read response from peer %s: %v\n", peerID, err)
		} else {
			fmt.Printf("PingCommand: Received response from peer %s: %s\n", peerID, strings.TrimSpace(response))
		}
	}
}

// Response for a ping command being sent to this host - Response with a Pong! - Notice how we are not sending another command back, just text
func (p *PingCommand) Respond(server *BootstrapServer, sendingStream network.Stream) {
	_, err := sendingStream.Write([]byte("Pong!\n"))
	if err != nil {
		log.Printf("PingResponse: Failed to send ping to peer %s: %v", sendingStream.Conn().RemotePeer(), err)
	} else {
		fmt.Printf("PingResponse: Sent pong to peer %s", sendingStream.Conn().RemotePeer())
	}
}

// Request the shared P and Q every peer should have in the network - all peers will request this from the host each round
type PQRequestCommand struct{}
func (pq *PQRequestCommand) Execute(server *BootstrapServer) {
	// Create a new stream to the peer
	stream, err := server.host.NewStream(context.Background(), server.sessionHost, protocolID)
	if err != nil {
		log.Printf("PQRequest: Failed to create stream to host %s: %v\n", server.sessionHost, err)
		return
	}
	defer stream.Close()

	_, err = stream.Write([]byte("CMDpqrequest\n"))
	if err != nil {
		log.Printf("PQRequest: Failed to send command to host %s: %v\n", server.sessionHost, err)
	} else {
		fmt.Printf("PQRequest: Sent command to host %s\n", server.sessionHost)
	}

	// Read the response - we do this as they won't be sending an actual *command* back, just some text
	responseBytes, err := io.ReadAll(stream)
	if err != nil {
		log.Printf("PQRequest: Failed to read response from host %s: %v\n", server.sessionHost, err)
	} else {
		pq := strings.Split(string(responseBytes), "\n")
		fmt.Printf("PQRequest: Received response from host: %s P: %s -- Q: %s\n", server.sessionHost, pq[0], pq[1])
		server.keyring.SetPQ(pq[0], pq[1]) // Set this servers p and q
		server.keyring.GenerateGlobalKeys()
	}
}

// Called on a 'CMDpqrequest' command call
func (pq *PQRequestCommand) Respond(server *BootstrapServer, sendingStream network.Stream) {
	PQData := server.keyring.GetPQString()
	_, err := sendingStream.Write([]byte(PQData))
	if err != nil {
		log.Printf("PQRequestResponse: Failed to send PQ to peer %s: %v", sendingStream.Conn().RemotePeer(), err)
	} else {
		fmt.Printf("PQRequestResponse: Sent PQ to peer %s", sendingStream.Conn().RemotePeer())
	}
}


// Testing function - This will take a message, and send to each peer for encryption and decryption
type TestEncryptionCommand struct {
	Message string
}

// Execute sends the message to all peers for encryption and ensures commutative encryption correctness.
func (t *TestEncryptionCommand) Execute(server *BootstrapServer) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	// Hash the original message
	originalMessage := sra.HashMessage(t.Message)

	fmt.Printf("Original message hash: %s\n", originalMessage.String())

	currentMessage := server.keyring.EncryptWithGlobalKeys(originalMessage)
	fmt.Printf("Host-encrypted hash: %s\n", currentMessage.String())

	// Encrypt the message with each peer
	for peerID := range server.peers {
		if peerID == server.host.ID() {
			continue // Skip self
		}

		// Send the message to the peer for encryption
		encryptedMessage, err := sendForEncryption(server, peerID, currentMessage)
		if err != nil {
			log.Printf("Failed to encrypt message with peer %s: %v\n", peerID, err)
			return
		}
		fmt.Printf("Message after encryption by peer %s: %s\n", peerID, encryptedMessage.String())
		currentMessage = encryptedMessage
	}
	
	// Remove my encryption (to test commutativity)
	currentMessage = server.keyring.DecryptWithGlobalKeys(currentMessage)
	fmt.Printf("Host-decrypted hash: %s\n", currentMessage.String())

	// Decrypt the message with each peer in reverse order
	for peerID := range server.peers {
		if peerID == server.host.ID() {
			continue // Skip self
		}

		// Send the message to the peer for decryption
		decryptedMessage, err := sendForDecryption(server, peerID, currentMessage)
		if err != nil {
			log.Printf("Failed to decrypt message with peer %s: %v\n", peerID, err)
			return
		}
		fmt.Printf("Message after decryption by peer %s: %s\n", peerID, decryptedMessage.String())
		currentMessage = decryptedMessage
	}


	// Check if the final decrypted message matches the original
	if currentMessage.Cmp(originalMessage) == 0 {
		fmt.Println("Test passed: Original and decrypted messages match.")
	} else {
		fmt.Printf("Test failed: Original message %s and decrypted message %s do not match.\n",
			originalMessage.String(), currentMessage.String())
	}
}

// Respond not implemented for this command as it's not invoked remotely.
func (t *TestEncryptionCommand) Respond(server *BootstrapServer, stream network.Stream) {
	// Not needed for this test command
}


// Helper function for test
func sendForEncryption(server *BootstrapServer, peerID peer.ID, message *big.Int) (*big.Int, error) {
	// Open a new stream to the peer
	stream, err := server.host.NewStream(context.Background(), peerID, protocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %v", err)
	}
	defer stream.Close()

	// Send the "ENCRYPT" command along with the message
	_, err = stream.Write([]byte(fmt.Sprintf("CMDencrypt %s\n", message.String())))
	if err != nil {
		return nil, fmt.Errorf("failed to send encryption command: %v", err)
	}

	// Read the response
	reader := bufio.NewReader(stream)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read encryption response: %v", err)
	}
	response = strings.TrimSpace(response)

	// Parse the encrypted message
	encryptedMessage := new(big.Int)
	encryptedMessage.SetString(response, 10)
	return encryptedMessage, nil
}

// Helper function for test
func sendForDecryption(server *BootstrapServer, peerID peer.ID, message *big.Int) (*big.Int, error) {
	// Open a new stream to the peer
	stream, err := server.host.NewStream(context.Background(), peerID, protocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %v", err)
	}
	defer stream.Close()

	// Send the "DECRYPT" command along with the message
	_, err = stream.Write([]byte(fmt.Sprintf("CMDdecrypt %s\n", message.String())))
	if err != nil {
		return nil, fmt.Errorf("failed to send decryption command: %v", err)
	}

	// Read the response
	reader := bufio.NewReader(stream)
	response, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read decryption response: %v", err)
	}
	response = strings.TrimSpace(response)

	// Parse the decrypted message
	decryptedMessage := new(big.Int)
	decryptedMessage.SetString(response, 10)
	return decryptedMessage, nil
}
