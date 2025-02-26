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

	// Player nicknames tied to their peer.ID - Handled by network
	Players map[peer.ID]string

	// Bets made during this round
	BetHistory map[peer.ID]float64

	// On play being pressed
	PlayersMoney map[peer.ID]float64 // Players money by peer ID

	// Turn order
	TurnOrder map[int]peer.ID // Handled by network (based off of candidate list)
	WhosTurn  int

	// Round variables
	// Holds who has folded this round
	FoldedPlayers map[peer.ID]bool
	// Bets made during this phase - used for raising, call, and check
	PhaseBets map[peer.ID]float64
	MylastBet float64 // Holds the last bet I placed
	// Holds weather a player has played this phase - Used to determine when the move to next phase
	PlayedThisPhase map[peer.ID]bool

	// Table rules (set my host)
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
	gs.WhosTurn = 0
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
	gs.FoldedPlayers[peerID] = false
	gs.PlayedThisPhase[peerID] = false
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

	delete(gs.BetHistory, peerID)
	delete(gs.FoldedPlayers, peerID)

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

	for i := 0; i < len(IDs); i++ {
		gs.TurnOrder[i] = IDs[i]
	}
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
	var whosTurn string

	for i := 0; i < len(gs.TurnOrder); i++ { // We disregard any 'exists' stuff as by this point we have already locked in everyone
		peerID := gs.TurnOrder[i]
		peerNickname := gs.Players[peerID]
		peerMoney := gs.PlayersMoney[peerID]
		players = append(players, peerNickname)
		money = append(money, peerMoney)
		if peerID == gs.Me {
			me = peerNickname
		}
		if i == gs.WhosTurn {
			whosTurn = peerNickname
		}
	}

	return channelmanager.PlayerInfo{Players: players, Money: money, Me: me, HighestBet: gs.GetHighestbetThisPhase(), WhosTurn: whosTurn}
}

func (gs *GameState) GetHighestbetThisPhase() float64 {
	highestBet := 0.0
	for peerID := range gs.Players {
		if highestBet < gs.PhaseBets[peerID] {
			highestBet = gs.PhaseBets[peerID]
		}
	}
	return highestBet
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

func (gs *GameState) PlayerBet(peerID peer.ID, bet float64) {
	gs.mu.Lock()
	gs.PlayersMoney[peerID] -= bet // Lower players money
	gs.BetHistory[peerID] += bet   // Update the best history for the round
	gs.PhaseBets[peerID] += bet
	gs.MylastBet = bet
	gs.mu.Unlock()
}

func (gs *GameState) PlayerRaise(peerID peer.ID, bet float64) {
	gs.PlayerBet(peerID, bet)
	for key := range gs.PlayedThisPhase {
		if gs.PlayedThisPhase[key] {
			gs.PlayedThisPhase[key] = false // We need to make the others call or raise or fold again
		}
	}

	if peerID == gs.Me {
		gs.PlayedThisPhase[gs.Me] = true
	}
}

func (gs *GameState) PlayerCall(peerID peer.ID) {
	gs.PlayerBet(peerID, gs.GetHighestbetThisPhase())
	gs.PlayedThisPhase[peerID] = true
}

func (gs *GameState) PlayerFold(peerID peer.ID) {
	gs.FoldedPlayers[peerID] = true
	delete(gs.PlayedThisPhase, peerID) // As he won't be apart of anymore phases
}

// Determines whos going and will check if the phases need to be switched
func (gs *GameState) NextTurn() {
	if len(gs.Players) == 1 { // If your last sitting at the table, just leave
		gs.EndRound()
	}

	phaseSwitch := true
	for key := range gs.PlayedThisPhase {
		if !gs.PlayedThisPhase[key] {
			phaseSwitch = false
		}
	}
	if phaseSwitch {
		gs.NextPhase()
	}

	// Modular arithmetic to wrap around
	nextValidPlayer := (gs.WhosTurn + 1) % len(gs.Players)

	for gs.FoldedPlayers[gs.TurnOrder[nextValidPlayer]] { // Skip all folded players
		nextValidPlayer = (nextValidPlayer + 1) % len(gs.Players)
	}

	gs.WhosTurn = nextValidPlayer

	channelmanager.TGUI_PotChan <- gs.GetCurrentPot()    // Updates the pot
	channelmanager.TGUI_PlayerInfo <- gs.GetPlayerInfo() // Updates the cards
}

func (gs *GameState) IsMyTurn() bool {
	return gs.Me == gs.TurnOrder[gs.WhosTurn]
}

func (gs *GameState) NextPhase() {
	gs.MylastBet = 0
	for key := range gs.PlayedThisPhase {
		gs.PlayedThisPhase[key] = false
	}
	for key := range gs.PhaseBets {
		gs.PhaseBets[key] = 0.0
	}

	// (e.g., "preflop", "flop", "turn", "river")
	switch gs.Phase {
	case "preflop":
		gs.Phase = "flop"
	case "flop":
		gs.Phase = "turn"
	case "turn":
		gs.Phase = "river"
	case "river":
		log.Println("Round over!")
		gs.EndRound()
	}
	fmt.Println("CURRENT PHASE: " + gs.Phase)
}

func (gs *GameState) PlayerCheck(peerID peer.ID) {
	gs.PlayedThisPhase[peerID] = true
}

// This will be called when all phases have been done, or if there is only 1 person left at the table
func (gs *GameState) EndRound() {
	channelmanager.TGUI_EndRound <- struct{}{}
}
