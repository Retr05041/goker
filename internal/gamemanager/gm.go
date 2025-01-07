package gamemanager

import (
	"fmt"
	"goker/internal/channelmanager"
	"goker/internal/gui"
	"goker/internal/p2p"

	"fyne.io/fyne/v2/canvas"
)

type GameEvent struct {
	EventType string
	State     *GameState
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

type GameManager struct {
	currentState *GameState
	network      *p2p.GokerPeer
}

func (gm *GameManager) StartGame(minBet *float64) {
	fmt.Println("Starting game!")
	gm.currentState = new(GameState)
	gm.currentState.StartGame(minBet)
	fmt.Println("State initialized!")

	// Init channels
	channelmanager.Init()
	go gm.listenForActions()
	fmt.Println("Channels initialized!")

	fmt.Println("Running GUI...")
	gui.Init()
}

// Listen for actions from the GUI (like button presses)
func (gm *GameManager) listenForActions() {
	for {
		select {
		case givenAction := <-channelmanager.ActionChannel:
			switch givenAction.Action {
			case "InitH&D":
				gm.initHand()
				gm.initBoard()
			case "Raise":
				// Handle raise action
				fmt.Println("Handling Raise action")
				// Update state accordingly
				gm.currentState.MyMoney -= *givenAction.Data
				gm.currentState.Pot += *givenAction.Data
				channelmanager.MyMoneyChannel <- gm.currentState.MyMoney
				channelmanager.PotChannel <- gm.currentState.Pot
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

// Setup hand with back of cards
func (gm *GameManager) initHand() {
	for i := 0; i < 2; i++ {
		cardImage := canvas.NewImageFromFile("media/svg_playing_cards/backs/png_96_dpi/red.png")
		cardImage.FillMode = canvas.ImageFillOriginal

		gm.currentState.MyHand = append(gm.currentState.MyHand, cardImage)
	}

	channelmanager.HandChannel <- gm.currentState.MyHand // Send the new hand through the channel
}

// Setup board with back of cards
func (gm *GameManager) initBoard() {
	for i := 0; i < 5; i++ {
		cardImage := canvas.NewImageFromFile("media/svg_playing_cards/backs/png_96_dpi/red.png")
		cardImage.FillMode = canvas.ImageFillOriginal

		gm.currentState.Board = append(gm.currentState.Board, cardImage)
	}

	channelmanager.BoardChannel <- gm.currentState.Board
}

// Load two cards into your hand and update the grid
func (gm *GameManager) loadHand(cardOneName, cardTwoName string) {
	cardOne := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardOneName + ".png")
	cardOne.FillMode = canvas.ImageFillOriginal

	cardTwo := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardTwoName + ".png")
	cardTwo.FillMode = canvas.ImageFillOriginal

	gm.currentState.MyHand[0] = cardOne
	gm.currentState.MyHand[1] = cardTwo
	channelmanager.HandChannel <- gm.currentState.MyHand
}

func (gm *GameManager) loadFlop(cardOneName, cardTwoName, cardThreeName string) {
	cardOne := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardOneName + ".png")
	cardOne.FillMode = canvas.ImageFillOriginal

	cardTwo := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardTwoName + ".png")
	cardTwo.FillMode = canvas.ImageFillOriginal

	cardThree := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardThreeName + ".png")
	cardThree.FillMode = canvas.ImageFillOriginal

	gm.currentState.Board[0] = cardOne
	gm.currentState.Board[1] = cardTwo
	gm.currentState.Board[2] = cardThree
	channelmanager.BoardChannel <- gm.currentState.Board
}

func (gm *GameManager) loadTurn(cardName string) {
	card := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardName + ".png")
	card.FillMode = canvas.ImageFillOriginal

	gm.currentState.Board[3] = card
	channelmanager.BoardChannel <- gm.currentState.Board
}

func (gm *GameManager) loadRiver(cardName string) {
	card := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardName + ".png")
	card.FillMode = canvas.ImageFillOriginal

	gm.currentState.Board[4] = card
	channelmanager.BoardChannel <- gm.currentState.Board
}
