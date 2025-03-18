package gamemanager

import (
	"fmt"
	"goker/internal/channelmanager"
	"goker/internal/gamestate"
	"goker/internal/gui"
	"goker/internal/p2p"
	"log"
	"time"

	"fyne.io/fyne/v2/canvas"
	"github.com/libp2p/go-libp2p/core/peer"
)

type GameManager struct {
	state   *gamestate.GameState // State built by the host and network
	network *p2p.GokerPeer

	MyNickname string
	MyHand     []*canvas.Image // Cards for the current player (images for the GUI to render)
	Board      []*canvas.Image // Community cards on the board (images for the gui to render)

	stopTurnTimer chan struct{}
}

func (gm *GameManager) StartGame() {
	// Init channels
	channelmanager.Init()

	go gm.listenForActions()

	go gm.phaseListener()

	go gm.roundChanger()

	gui.Init()
}

// Listen for actions from the GUI (like button presses)
func (gm *GameManager) listenForActions() {
	for {
		select {
		case givenAction := <-channelmanager.FGUI_ActionChan:
			switch givenAction.Action {
			case "Init": // Initialise everything
				gm.initBoard()
			case "hostOrConnectPressed": // Weather you are hosting or connecting this is called
				// Setup network node
				gm.network = new(p2p.GokerPeer)

				// Setup gamestate
				gm.state = new(gamestate.GameState)
				gm.state.Players = make(map[peer.ID]string)
				gm.state.PlayersMoney = make(map[peer.ID]float64)
				gm.state.BetHistory = make(map[peer.ID]float64)
				gm.state.PhaseBets = make(map[peer.ID]float64)
				gm.state.TurnOrder = make(map[int]peer.ID)
				gm.state.FoldedPlayers = make(map[peer.ID]bool)
				gm.state.PlayedThisPhase = make(map[peer.ID]bool)

				if len(givenAction.DataS) == 1 {
					go gm.network.Init(givenAction.DataS[0], true, "", gm.state) // Hosting
				} else {
					go gm.network.Init(givenAction.DataS[0], false, givenAction.DataS[1], gm.state) // Connecting
				}
				<-channelmanager.FNET_NetActionDoneChan                                                                 // Wait for network to be done setting up
				channelmanager.TGUI_AddressChan <- []string{gm.network.ThisHostLBAddress, gm.network.ThisHostLNAddress} // Tell the GUI the addresses we need
			case "startRound": // TODO: This action should gather table rules for the state
				gm.network.SetTurnOrderWithLobby()                 // Sets the turn order
				gm.state.FreshState(nil, nil)                      // Initialize table settings after the lobby is populated
				gm.network.ExecuteCommand(&p2p.InitTableCommand{}) // Tells others table rules and solidify the state

				// Fill cards in GUI
				channelmanager.TGUI_PlayerInfo <- gm.state.GetPlayerInfo()

				gm.RunProtocol()

				gm.network.ExecuteCommand(&p2p.MoveToTableCommand{}) // Tell everyone to move to the game table

				// If it's my turn, start the turn timer
				if gm.state.IsMyTurn() {
					gm.startTurnTimer()
				}
			case "Raise":
				if !gm.state.IsMyTurn() {
					fmt.Println("Not your turn yet!")
					continue // NO BREAKING
				}
				gm.stopTurnTimerIfRunning() // Stop the auto-fold timer

				// Handle raise action
				fmt.Println("Handling Raise action")
				// Update state
				gm.state.PlayerRaise(gm.state.Me, givenAction.DataF)
				// Send to others
				gm.network.ExecuteCommand(&p2p.RaiseCommand{})

				gm.state.NextTurn()
				if gm.state.IsMyTurn() {
					gm.startTurnTimer()
				}
			case "Call":
				if !gm.state.IsMyTurn() {
					fmt.Println("Not your turn yet!")
					continue
				}
				gm.stopTurnTimerIfRunning()

				// Handle call action
				fmt.Println("Handling Call action")
				// Update state
				gm.state.PlayerCall(gm.state.Me)
				// Others
				gm.network.ExecuteCommand(&p2p.CallCommand{})

				gm.state.NextTurn()
				if gm.state.IsMyTurn() {
					gm.startTurnTimer()
				}
			case "Check":
				if !gm.state.IsMyTurn() {
					fmt.Println("Not your turn yet!")
					continue
				}
				gm.stopTurnTimerIfRunning()

				// Handle call action
				fmt.Println("Handling Check action")
				gm.state.PlayerCheck(gm.state.Me)
				// Others
				gm.network.ExecuteCommand(&p2p.CheckCommand{})

				gm.state.NextTurn() // Skip your turn for now
				if gm.state.IsMyTurn() {
					gm.startTurnTimer()
				}
			case "Fold":
				if !gm.state.IsMyTurn() {
					fmt.Println("Not your turn yet!")
					continue
				}
				gm.stopTurnTimerIfRunning()

				// Handle fold action
				fmt.Println("Handling Fold action")
				gm.state.PlayerFold(gm.state.Me)              // I fold
				gm.network.ExecuteCommand(&p2p.FoldCommand{}) // Tell others I have folded

				gm.state.NextTurn() // Move to next person
				if gm.state.IsMyTurn() {
					gm.startTurnTimer()
				}
			}
		}
	}
}

func (gm *GameManager) phaseListener() {
	for {
		<-channelmanager.TGM_PhaseCheck

		gm.state.MyBet = 0

		for id := range gm.state.PlayedThisPhase {
			if !gm.state.FoldedPlayers[id] { // Skip any folded players
				gm.state.PlayedThisPhase[id] = false
			}
		}

		for id := range gm.state.PhaseBets {
			gm.state.PhaseBets[id] = 0.0
		}

		switch gm.state.Phase {
		case "preflop":
			gm.state.Phase = "flop"
			gm.network.ExecuteCommand(&p2p.RequestFlop{})
			gm.network.ExecuteCommand(&p2p.PushTagCommand{}) // Update tag for next phase
			fmt.Println("CURRENT PHASE: " + gm.state.Phase)
		case "flop":
			gm.state.Phase = "turn"
			gm.network.ExecuteCommand(&p2p.RequestTurn{})
			gm.network.ExecuteCommand(&p2p.PushTagCommand{})
			fmt.Println("CURRENT PHASE: " + gm.state.Phase)
		case "turn":
			gm.state.Phase = "river"
			gm.network.ExecuteCommand(&p2p.RequestRiver{})
			gm.network.ExecuteCommand(&p2p.PushTagCommand{})
			fmt.Println("CURRENT PHASE: " + gm.state.Phase)
		case "river":
			log.Println("Round over! Determining winner and starting new round!")
			gm.network.ExecuteCommand(&p2p.RequestOthersHands{})
			gm.state.EndRound()
		}

		channelmanager.TGS_PhaseSwitchDone <- struct{}{} // Continue with the next turn function in GS
	}
}

func (gm *GameManager) roundChanger() {
	for {
		<-channelmanager.TGM_EndRound
		gm.EvaluateHands()
	}
}

func (gm *GameManager) startTurnTimer() {
	// Create a channel to stop the timer if the player makes a move
	stopTimer := make(chan struct{})

	// Start a goroutine for the timer
	go func() {
		select {
		case <-time.After(30 * time.Second):
			if gm.state.IsMyTurn() {
				fmt.Println("Time's up! Auto-folding...")
				gm.state.PlayerFold(gm.state.Me)              // Fold the player
				gm.network.ExecuteCommand(&p2p.FoldCommand{}) // Notify others
				gm.state.NextTurn()                           // Move to the next player's turn
			}
		case <-stopTimer:
			// The player made a move in time, so we stop the timer
			return
		}
	}()

	// Store the stop channel in GameManager to stop the timer when needed
	gm.stopTurnTimer = stopTimer
}

// Call this function inside your existing action cases (Raise, Call, Check, Fold) to stop the timer when an action is taken:
func (gm *GameManager) stopTurnTimerIfRunning() {
	if gm.stopTurnTimer != nil {
		close(gm.stopTurnTimer) // Stops the timer goroutine
		gm.stopTurnTimer = nil  // Reset it so it's not used again
	}
}
