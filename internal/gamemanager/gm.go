package gamemanager

import (
	"fmt"
	"goker/internal/channelmanager"
	"goker/internal/gamestate"
	"goker/internal/gui"
	"goker/internal/p2p"

	"fyne.io/fyne/v2/canvas"
)

type GameManager struct {
	state   gamestate.GameState
	network *p2p.GokerPeer

	MyNickname string
	MyHand []*canvas.Image // Cards for the current player (images for the GUI to render)
	Board  []*canvas.Image // Community cards on the board (images for the gui to render)
}

func (gm *GameManager) StartGame() {
	// Init channels
	channelmanager.Init()
	go gm.listenForActions()

	gui.Init()
}


// Listen for actions from the GUI (like button presses)
func (gm *GameManager) listenForActions() {
	for {
		select {
		case givenAction := <-channelmanager.FGUI_ActionChan:
			switch givenAction.Action {
			case "Init":
				gm.initHand()
				gm.initBoard()
				channelmanager.TGUI_MyMoneyChan <- gm.state.StartingCash
				channelmanager.TGUI_PotChan <- gm.state.Pot
			case "hostOrConnectPressed":
				fmt.Println("Got here!")
				gm.network = new(p2p.GokerPeer)
				if givenAction.DataS == nil {
					go gm.network.Init(true, "")
				} else {
					go gm.network.Init(false, *givenAction.DataS)
				}
				gm.state = <- channelmanager.TFNET_GameStateChan // Wait for network to be done setting up, should give us a new state
				channelmanager.TGUI_AddressChan <- []string{gm.network.ThisHostLBAddress, gm.network.ThisHostLNAddress}
			case "startRound":
				gm.network.ExecuteCommand(&p2p.StartRoundCommand{})
			case "Raise":
				// Handle raise action
				fmt.Println("Handling Raise action")
				// Update state
				gm.state.PlayersMoney[gm.MyNickname] -= *givenAction.DataF
				gm.state.Pot += *givenAction.DataF
				gm.state.BetHistory[gm.MyNickname] = *givenAction.DataF // Update state

				// Update GUI
				channelmanager.TGUI_MyMoneyChan <- gm.state.PlayersMoney[gm.MyNickname]
				channelmanager.TGUI_PotChan <- gm.state.Pot

				// Send newly updated state to the network for processing
				//channelmanager.TFNET_GameStateChan <- gm.state 
				//gm.network.ExecuteCommand(&p2p.RaiseCommand{}) // Update peers about raise
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
