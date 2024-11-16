package p2p

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"log"

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
