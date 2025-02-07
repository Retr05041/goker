package gamemanager

import (
	"fmt"
	"goker/internal/channelmanager"
	"goker/internal/gamestate"
	"goker/internal/gui"
	"goker/internal/p2p"

	"fyne.io/fyne/v2/canvas"
	"github.com/libp2p/go-libp2p/core/peer"
)

type GameManager struct {
	state   *gamestate.GameState // State built by the host and network
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
			case "Init": // Initialise everything
				gm.initHand()
				gm.initBoard()
				gm.state = new(gamestate.GameState)
			case "hostOrConnectPressed": // Weather you are hosting or connecting this is called
				// Setup network node
				gm.network = new(p2p.GokerPeer)

				// Setup gamestate
				gm.state = new(gamestate.GameState)
				gm.state.Players = make(map[peer.ID]string)
				gm.state.PlayersMoney = make(map[peer.ID]float64)
				gm.state.BetHistory = make(map[peer.ID]float64)
				gm.state.TurnOrder = make(map[int]peer.ID)

				if len(givenAction.DataS) == 1 {
					go gm.network.Init(givenAction.DataS[0], true, "", gm.state) // Hosting
				} else {
					go gm.network.Init(givenAction.DataS[0], false, givenAction.DataS[1], gm.state) // Connecting
				}
				<- channelmanager.FNET_NetActionDoneChan // Wait for network to be done setting up
				channelmanager.TGUI_AddressChan <- []string{gm.network.ThisHostLBAddress, gm.network.ThisHostLNAddress} // Tell the GUI the addresses we need
			case "startRound": // TODO: This action should gather table rules for the state
				// Refresh state 
				gm.state.FreshState(nil, nil) // Initialise the state for this round (doesn't affect turn order or peers etc.)

				channelmanager.TNET_ActionChan <- channelmanager.ActionType{Action: "startround"} // Tell network to populate the state with everyone connected

				// TODO: Send Player info to GUI before letting the GUI progress to the round screen - Player nicknames and their money

				<- channelmanager.FNET_NetActionDoneChan // wait for network to be finished with the startround command

				// Fill the GUI with populated state
				channelmanager.TGUI_PlayerInfo <- gm.state.GetPlayerInfo()

				channelmanager.TGUI_StartRound <- struct{}{} // Tell GUI to move to the table UI
			case "Raise":
				// Handle raise action
				fmt.Println("Handling Raise action")

				// Update state locally
				gm.state.PlayersMoney[gm.network.ThisHost.ID()] -= givenAction.DataF
				gm.state.BetHistory[gm.network.ThisHost.ID()] += givenAction.DataF 

				// Update GUI locally
				channelmanager.TGUI_PotChan <- gm.state.GetCurrentPot()
				channelmanager.TGUI_PlayerInfo <- gm.state.GetPlayerInfo()

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
