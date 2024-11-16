package p2p

import (
	"fmt"
	"log"
	"context"
)

type Command interface {
	Execute(server *BootstrapServer)
}

// Execute a command on the server
func (s *BootstrapServer) ExecuteCommand(command Command) {
	command.Execute(s)
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
	}
}

// Response for a ping command being sent to this host - Response with a Pong!
type PingResponse struct {}
func (p *PingResponse) Execute(server *BootstrapServer) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	for peerID := range server.peers {
		if peerID == server.host.ID() {
			continue // Skip self
		}

		// Create a new stream to the peer
		stream, err := server.host.NewStream(context.Background(), peerID, protocolID)
		if err != nil {
			log.Printf("PingResponse: Failed to create stream to peer %s: %v", peerID, err)
			continue
		}
		defer stream.Close()

		// Send the "ping"
		_, err = stream.Write([]byte("Pong!\n"))
		if err != nil {
			log.Printf("PingResponse: Failed to send ping to peer %s: %v", peerID, err)
		} else {
			fmt.Printf("PingResponse: Sent pong to peer %s", peerID)
		}
	}
}
