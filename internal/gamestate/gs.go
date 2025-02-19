package gamestate

import (
	"fmt"
	"goker/internal/channelmanager"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

type GameState struct {
	// Gamestate mutex due to the network and gamemanager using the same state
	mu sync.Mutex
	Me peer.ID
	// Handled by network (built dynamically and 'should be' signed by all peers for validity)
	// On peer connection
	Players    map[peer.ID]string  // Player nicknames tied to their peer.ID
	BetHistory map[peer.ID]float64 // A map to store bets placed on the current round
	LastBet    float64             // Holds the last bet played (will be passed around for raising and possibly calling)

	// On play being pressed
	PlayersMoney map[peer.ID]float64 // Players money by peer ID
	TurnOrder    map[int]peer.ID     // Handled by network (based off of candidate list)
	WhosTurn     int

	// Handled by host
	StartingCash float64 // Starting cash for all players
	MinBet       float64 // Minimum bet required for the round (again from table settings)
	Phase        string  // Current phase of the game (e.g., "preflop", "flop", "turn", "river")
}

// Refresh state for new possible rounds
func (gs *GameState) FreshState(startingCash *float64, minBet *float64) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.Phase = "preflop"

	gs.StartingCash = 100.0
	if startingCash != nil {
		gs.StartingCash = *startingCash
	}
	for id := range gs.Players {
		gs.PlayersMoney[id] = gs.StartingCash
	}

	gs.MinBet = 1.0
	if minBet != nil {
		gs.MinBet = *minBet
	}

	gs.Phase = "preflop"
	gs.WhosTurn = 1 // left of the dealer, aka the host for the first round
}

// Function used by network for setting table rules from host
func (gs *GameState) FreshStateFromPayload(payload string) {
	payloadSplit := strings.Split(payload, "\n")

	startingCash, err := strconv.ParseFloat(payloadSplit[0], 64) // Parse as a 64 bit float
	if err != nil {
		log.Println(err)
	}
	minBet, err := strconv.ParseFloat(payloadSplit[1], 64)
	if err != nil {
		log.Println(err)
	}

	gs.FreshState(&startingCash, &minBet)
}

// For adding a new peer to the state
func (gs *GameState) AddPeerToState(peerID peer.ID, nickname string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if _, v := gs.Players[peerID]; v {
		log.Println("AddPeerToState: Peer already in state")
		return
	}
	gs.Players[peerID] = nickname
	gs.BetHistory[peerID] = 0.0
}

func (gs *GameState) RemovePeerFromState(peerID peer.ID) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	_, exists := gs.Players[peerID]
	if !exists {
		log.Println("RemovePeerFromState: Peer not in state")
		return
	}
	delete(gs.Players, peerID)
	// Remove bet history
	delete(gs.BetHistory, peerID)

	_, exists = gs.PlayersMoney[peerID]
	if !exists {
		log.Println("RemovePeerFromState: Peer doesn't have money")
		return
	}
	delete(gs.PlayersMoney, peerID)
}

// Get the current pot from the rounds bet history
func (gs *GameState) GetCurrentPot() float64 {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	pot := 0.0
	for _, b := range gs.BetHistory {
		pot += b
	}
	return pot
}

// Sets turn order Whoever is first in the incoming slice is put in the last place, assuming they are the dealer for this round
func (gs *GameState) SetTurnOrder(IDs []peer.ID) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if len(IDs) == 0 {
		return
	}

	hostID := IDs[0]

	for i := 1; i < len(IDs); i++ {
		gs.TurnOrder[i-1] = IDs[i]
	}

	gs.TurnOrder[len(IDs)-1] = hostID
}

// Check if a player exists
func (gs *GameState) PlayerExists(id peer.ID) bool {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if _, exists := gs.Players[id]; exists {
		return true
	}
	return false
}

// Get the nickname of a specific player
func (gs *GameState) GetNickname(id peer.ID) string {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	return gs.Players[id]
}

// Formatted player info to be sent to the GUI
func (gs *GameState) GetPlayerInfo() channelmanager.PlayerInfo {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	var players []string
	var money []float64
	var me string
	for i := 0; i < len(gs.TurnOrder); i++ { // We disregard any 'exists' stuff as by this point we have already locked in everyone
		peerID, _ := gs.TurnOrder[i]
		peerNickname, _ := gs.Players[peerID]
		peerMoney, _ := gs.PlayersMoney[peerID]
		players = append(players, peerNickname)
		money = append(money, peerMoney)
		if peerID == gs.Me {
			me = peerNickname
		}
	}

	return channelmanager.PlayerInfo{Players: players, Money: money, Me: me}
}

// Package up the table rules to be sent to others
func (gs *GameState) GetTableRules() string {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	var tableRules string
	tableRules += fmt.Sprintf("%.0f\n", gs.StartingCash)
	tableRules += fmt.Sprintf("%.0f\n", gs.MinBet)

	return tableRules
}

func (gs *GameState) GetTurnOrderIndex(peer peer.ID) *int {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	for i, v := range gs.TurnOrder {
		if v == peer {
			return &i
		}
	}
	return nil
}

func (gs *GameState) GetNumberOfPlayers() int {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	return len(gs.Players)
}

// Returns all ID's in turn order order
func (gs *GameState) GetTurnOrder() []peer.ID {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	var ids []peer.ID
	for _, v := range gs.TurnOrder {
		ids = append(ids, v)
	}

	return ids
}

// Update state based on players bet
func (gs *GameState) PlayerBet(peerID peer.ID, bet float64) {
	gs.mu.Lock()

	gs.LastBet = bet
	gs.PlayersMoney[peerID] -= bet
	gs.BetHistory[peerID] += bet

	gs.mu.Unlock()

	channelmanager.TGUI_PotChan <- gs.GetCurrentPot()    // Updates the pot
	channelmanager.TGUI_PlayerInfo <- gs.GetPlayerInfo() // Updates the cards
}

func (gs *GameState) GetLastBet() float64 {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return gs.LastBet
}

func (gs *GameState) DumpState() {
	fmt.Println("DUMPING STATE")
	fmt.Println("---------------------------------")
	log.Print("Players: ")
	for id, val := range gs.Players {
		fmt.Println(id.String() + ", " + val)
	}
	fmt.Println("---------")
	log.Print("PlayersMoney: ")
	for id, val := range gs.PlayersMoney {
		fmt.Printf("%s, %.1f\n", id, val)

	}
	fmt.Println("---------")
	log.Print("BetHistory: ")
	for id, val := range gs.BetHistory {
		fmt.Printf("%s, %.1f\n", id, val)

	}
	fmt.Println("---------")
	log.Print("TurnOrder: ")
	for i := 0; i < len(gs.TurnOrder); i++ {
		id, _ := gs.TurnOrder[i]
		fmt.Printf("%d. %s\n", i, id.String())
	}
	fmt.Println("---------")
	log.Printf("WhosTurn: %d", gs.WhosTurn)
	fmt.Println("---------")
	log.Printf("StartingCash: %.1f", gs.StartingCash)
	fmt.Println("---------")
	log.Printf("Pot: %.1f", gs.GetCurrentPot())
	fmt.Println("---------")
	log.Printf("MineBet: %.1f", gs.MinBet)
	fmt.Println("---------")
	log.Printf("Phase: %s", gs.Phase)
	fmt.Println("--------------------------------")
}
