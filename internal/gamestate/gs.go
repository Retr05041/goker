package gamestate

import (
	"fmt"
	"goker/internal/channelmanager"
	"log"
	"sort"
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
	MyBet     float64
	// Holds weather a player has played this phase - Used to determine when the move to next phase
	PlayedThisPhase map[peer.ID]bool

	// Table rules (set by host)
	StartingCash float64 // Starting cash for all players
	MinBet       float64 // Minimum bet required for the round (again from table settings)
	Phase        string  // Current phase of the game (e.g., "preflop", "flop", "turn", "river")

	// Winning player of previous round
	Winner             peer.ID
	SomeoneLeft        bool // Boolean for if someone leaves and hasn't folded yet
	NumOfPuzzlesBroken int  // this should go up by 1 with every time locked puzzle broken -
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

// When a player leaves, the round will change
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
	delete(gs.PhaseBets, peerID)
	delete(gs.PlayedThisPhase, peerID)
	delete(gs.PlayersMoney, peerID)

	// Remove peer from turn order and reindex turn order
	newTurnOrder := make(map[int]peer.ID)
	newIndex := 0
	for i := 0; i < len(gs.TurnOrder); i++ {
		if gs.TurnOrder[i] != peerID {
			newTurnOrder[newIndex] = gs.TurnOrder[i]
			newIndex++
		}
	}
	gs.TurnOrder = newTurnOrder

	// Adjust current turn if necessary
	if len(gs.TurnOrder) > 0 {
		gs.WhosTurn = gs.WhosTurn % len(gs.TurnOrder)
	} else {
		gs.WhosTurn = 0
	}
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

	return channelmanager.PlayerInfo{Players: players, Money: money, Me: me, HighestBet: gs.GetHighestbetThisPhase(), WhosTurn: whosTurn, MyBetsForThisPhase: gs.MyBet}
}

// GetHighestBetThisPhase will return either the highest someones bet this phase, or 0 if all bets are the same
func (gs *GameState) GetHighestbetThisPhase() float64 {
	var highestBet float64
	allEqual := true
	var firstBet float64
	first := true

	for peerID := range gs.Players {
		bet := gs.PhaseBets[peerID]

		if first {
			firstBet = bet
			first = false
		} else if bet != firstBet {
			allEqual = false
		}

		if bet > highestBet {
			highestBet = bet
		}
	}

	if allEqual {
		return 0
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

	// Extract keys (turn positions)
	positions := make([]int, 0, len(gs.TurnOrder))
	for pos := range gs.TurnOrder {
		positions = append(positions, pos)
	}

	// Sort keys to maintain turn order
	sort.Ints(positions)

	// Collect peer IDs in sorted order
	ids := make([]peer.ID, len(positions))
	for i, pos := range positions {
		ids[i] = gs.TurnOrder[pos]
	}

	return ids
}

func (gs *GameState) PlayerBet(peerID peer.ID, bet float64) {
	gs.mu.Lock()
	gs.PlayersMoney[peerID] -= bet // Lower players money
	gs.BetHistory[peerID] += bet   // Update the best history for the round
	gs.PhaseBets[peerID] += bet
	gs.mu.Unlock()
}

func (gs *GameState) PlayerRaise(peerID peer.ID, bet float64) {
	gs.PlayerBet(peerID, bet)
	for id := range gs.PlayedThisPhase {
		gs.PlayedThisPhase[id] = false // We need to make the others call or raise or fold again
	}

	if peerID == gs.Me {
		gs.MyBet += bet
	}
	gs.PlayedThisPhase[peerID] = true // Person who raised did in fact play this round
}

func (gs *GameState) PlayerCall(peerID peer.ID) {
	gs.PlayerBet(peerID, gs.GetHighestbetThisPhase())
	gs.MyBet = gs.GetHighestbetThisPhase()
	gs.PlayedThisPhase[peerID] = true
}

func (gs *GameState) PlayerFold(peerID peer.ID) {
	gs.FoldedPlayers[peerID] = true
}

// Determines whos going and will check if the phases need to be switched
func (gs *GameState) NextTurn() {
	// Check if only one player is left at the table (ignore disconnected players)
	nonFoldedCount := 0

	for id := range gs.Players {
		if !gs.FoldedPlayers[id] {
			nonFoldedCount++
		}
	}

	// If only one non-folded player remains, end the round immediately
	if nonFoldedCount == 1 {
		log.Println("Only one player remains, ending the round...")
		gs.EndRound()
		return
	}

	// If someone left, ensure we properly handle ending the round at the right time
	if gs.SomeoneLeft {
		log.Println("Someone left, checking if we need to end the round...")
		if nonFoldedCount == 1 {
			gs.EndRound()
			return
		}
	}

	// From all players who haven't folded, if there are any
	// that haven't played OR haven't matched the highest bet so far, don't switch the phase
	phaseSwitch := true
	highestBetThisPhase := gs.GetHighestbetThisPhase()
	for id := range gs.PlayedThisPhase {
		if !gs.FoldedPlayers[id] {
			if !gs.PlayedThisPhase[id] || gs.PhaseBets[id] < highestBetThisPhase {
				phaseSwitch = false
			}
		}
	}
	if phaseSwitch {
		if gs.SomeoneLeft {
			log.Println("Someone left, ending round instead of changing phase...")
			gs.EndRound()
			return
		} else {
			channelmanager.TGM_PhaseCheck <- struct{}{} // Tell gm to switch phases
			<-channelmanager.TGS_PhaseSwitchDone
		}
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

func (gs *GameState) PlayerCheck(peerID peer.ID) {
	gs.PlayedThisPhase[peerID] = true
}

// The gamestate will call this so the game manager can reset everything and begin the next round
func (gs *GameState) EndRound() {
	channelmanager.TGM_EndRound <- struct{}{}
}

func (gs *GameState) Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
