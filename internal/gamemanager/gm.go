package gamemanager

import (
	"fmt"
	"goker/internal/channelmanager"
	"goker/internal/gamestate"
	"goker/internal/gui"
	"goker/internal/p2p"
	"log"

	"fyne.io/fyne/v2/canvas"
	"github.com/libp2p/go-libp2p/core/peer"
)

type GameManager struct {
	state   *gamestate.GameState // State built by the host and network
	network *p2p.GokerPeer

	MyNickname string
	MyHand     []*canvas.Image // Cards for the current player (images for the GUI to render)
	Board      []*canvas.Image // Community cards on the board (images for the gui to render)
}

func (gm *GameManager) StartGame() {
	// Init channels
	channelmanager.Init()

	go gm.listenForActions()

	go gm.phaseListener()

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

				// Setup keyring for this round
				gm.network.Keyring.GeneratePQ()
				gm.network.Keyring.GenerateKeys()

				// Setup deck
				gm.network.ExecuteCommand(&p2p.SendPQCommand{})            // Send everyone the generated P and Q so they can setup their Keyring
				gm.network.ExecuteCommand(&p2p.ProtocolFirstStepCommand{}) // Setting up deck pt.1
				gm.network.ExecuteCommand(&p2p.BroadcastNewDeck{})
				gm.network.ExecuteCommand(&p2p.ProtocolSecondStepCommand{}) // Setting up deck pt.2 & Sets everyones hands
				gm.network.ExecuteCommand(&p2p.BroadcastDeck{})

				// Setup hands
				gm.network.ExecuteCommand(&p2p.CanRequestHand{}) // Deals hands one player at at time
				gm.network.ExecuteCommand(&p2p.RequestHandCommand{})

				// TODO:
				//gm.network.ExecuteCommand(&p2p.KeyExchangeCommand{}) // Everyone sends each others timelocked payload to each other and they all begin to crack it

				gm.network.ExecuteCommand(&p2p.MoveToTableCommand{}) // Tell everyone to move to the game table
			case "Raise":
				if !gm.state.IsMyTurn() {
					fmt.Println("Not your turn yet!")
					continue // NO BREAKING
				}
				// Handle raise action
				fmt.Println("Handling Raise action")
				// Update state
				gm.state.PlayerRaise(gm.state.Me, givenAction.DataF)
				// Send to others
				gm.network.ExecuteCommand(&p2p.RaiseCommand{})

				gm.state.NextTurn()
			case "Call":
				if !gm.state.IsMyTurn() {
					fmt.Println("Not your turn yet!")
					continue
				}
				// Handle call action
				fmt.Println("Handling Call action")
				// Update state
				gm.state.PlayerCall(gm.state.Me)
				// Others
				gm.network.ExecuteCommand(&p2p.CallCommand{})

				gm.state.NextTurn()
			case "Check":
				if !gm.state.IsMyTurn() {
					fmt.Println("Not your turn yet!")
					continue
				}
				// Handle call action
				fmt.Println("Handling Check action")
				gm.state.PlayerCheck(gm.state.Me)
				// Others
				gm.network.ExecuteCommand(&p2p.CheckCommand{})

				gm.state.NextTurn() // Skip your turn for now
			case "Fold":
				if !gm.state.IsMyTurn() {
					fmt.Println("Not your turn yet!")
					continue
				}
				// Handle fold action
				fmt.Println("Handling Fold action")
				gm.state.PlayerFold(gm.state.Me)              // I fold
				gm.network.ExecuteCommand(&p2p.FoldCommand{}) // Tell others I have folded

				gm.state.NextTurn() // Move to next person
			}
		}
	}
}

func (gm *GameManager) phaseListener() {
	for {
		<-channelmanager.TGM_PhaseCheck
		gm.state.MyBet = 0
		for key := range gm.state.PlayedThisPhase {
			gm.state.PlayedThisPhase[key] = false
		}
		for key := range gm.state.PhaseBets {
			gm.state.PhaseBets[key] = 0.0
		}

		// (e.g., "preflop", "flop", "turn", "river")
		switch gm.state.Phase {
		case "preflop":
			gm.state.Phase = "flop"
			gm.network.ExecuteCommand(&p2p.RequestFlop{})
		case "flop":
			gm.state.Phase = "turn"
			gm.network.ExecuteCommand(&p2p.RequestTurn{})
		case "turn":
			gm.state.Phase = "river"
			gm.network.ExecuteCommand(&p2p.RequestRiver{})
		case "river":
			log.Println("Round over!")
			gm.state.EndRound()
		}
		fmt.Println("CURRENT PHASE: " + gm.state.Phase)
	}
}
