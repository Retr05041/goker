package gamemanager

import (
	"fmt"
	"fyne.io/fyne/v2/canvas"
	"goker/internal/channelmanager"
	"goker/internal/gui"
	"goker/internal/p2p"
)

type GameManager struct {
	state   *GameState
	network *p2p.GokerPeer
}

func (gm *GameManager) StartGame(minBet *float64) {
	gm.state = new(GameState)
	gm.state.StartGame(minBet)

	// Init channels
	channelmanager.Init()
	go gm.listenForActions()

	gui.Init()
}

type GameState struct {
	Pot        float64 // The current pot amount
	MyMoney    float64
	MinBet     float64            // Minimum bet required for the round
	MyHand     []*canvas.Image    // Cards for the current player (images for the GUI to render)
	Board      []*canvas.Image    // Community cards on the board (images for the gui to render)
	Phase      string             // Current phase of the game (e.g., "preflop", "flop", "turn", "river")
	BetHistory map[string]float64 // A map to store bets placed by players
}

// StartGame initializes the game state
func (gs *GameState) StartGame(minBet *float64) {
	gs.Phase = "preflop"
	gs.Pot = 0.0
	gs.MyMoney = 100.0
	if minBet == nil {
		gs.MinBet = 1.0
	} else {
		gs.MinBet = *minBet
	}
}

// Listen for actions from the GUI (like button presses)
func (gm *GameManager) listenForActions() {
	for {
		select {
		case givenAction := <-channelmanager.ActionChannel:
			switch givenAction.Action {
			case "Init":
				gm.initHand()
				gm.initBoard()
				channelmanager.MyMoneyChannel <- gm.state.MyMoney
				channelmanager.PotChannel <- gm.state.Pot
			case "Raise":
				// Handle raise action
				fmt.Println("Handling Raise action")
				// Update state accordingly
				gm.state.MyMoney -= *givenAction.Data
				gm.state.Pot += *givenAction.Data
				channelmanager.MyMoneyChannel <- gm.state.MyMoney
				channelmanager.PotChannel <- gm.state.Pot
			case "Fold":
				// Handle fold action
				fmt.Println("Handling Fold action")
				// Update state accordingly
			case "Call":
				// Handle call action
				fmt.Println("Handling Call action")
				// Update state accordingly
			case "Check":
				// Handle call action
				fmt.Println("Handling Call action")
				// Update state accordingly
			}
		}
	}
}
