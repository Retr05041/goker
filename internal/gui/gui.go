package gui

import (
	"fmt"
	"goker/internal/channelmanager"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var (
	// GUI Settings
	MAX_WIDTH  = 600 // 600
	MAX_HEIGHT = 400 //400

	// Colors
	BLUE = color.NRGBA{R: 0, G: 173, B: 216, A: 255}
)

// Runner for gui
func Init() {
	// Setup GUI
	myApp := app.New()
	mainWindow := myApp.NewWindow("Goker")

	// Init all scene elements
	initElements()

	// Listen for updated from GameManager
	go gmListener()

	// Init everything on the GM side
	channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "Init"}

	// Run the first scene
	showMenuUI(mainWindow)

	// Run GUI
	mainWindow.Resize(fyne.NewSize(float32(MAX_WIDTH), float32(MAX_HEIGHT))) // Set the window size
	mainWindow.ShowAndRun()
}

// Since we want to keep logic out of the GUI, this will listen for any updates from specific parts of the state
func gmListener() {
	for {
		select {
		case hand := <-channelmanager.TGUI_HandChan: // Hand = whatever is coming in from the handChannel
			updateHandImages(hand)
		case board := <-channelmanager.TGUI_BoardChan:
			updateBoardImages(board)
		case pot := <-channelmanager.TGUI_PotChan:
			updatePot(pot)
		case numOfPlayers := <-channelmanager.FNET_NumOfPlayersChan: // Gets it straight from the network - This is updated when new players join the lobby
			updateNumOfPlayers(numOfPlayers)
		case address := <- channelmanager.TGUI_AddressChan:
			updateAddress(address)
		case playerInfo := <- channelmanager.TGUI_PlayerInfo:
			updateCards(playerInfo)
		default:
			// Do nothing, stops blocking
		}
	}
}

func updateHandImages(hand []*canvas.Image) {
	fmt.Println("Updating Hand Images")

	newGrid := container.NewGridWithColumns(2)
	for _, image := range hand {
		newGrid.Add(image)
	}
	handGrid = container.NewGridWrap(handSize, newGrid)
	handGrid.Refresh()
}

func updateBoardImages(board []*canvas.Image) {
	fmt.Println("Updating Board Images")

	newGrid := container.NewGridWithColumns(5)
	for _, image := range board {
		newGrid.Add(image)
	}
	boardGrid= container.NewGridWrap(boardSize, newGrid)
	boardGrid.Refresh()
}

func updatePot(pot float64) {

	potLabel.SetText(fmt.Sprintf("Pot: %.0f", pot))
	potLabel.Refresh()
}

func updateStartingMoney(money float64) {
	moneyLabel.SetText(fmt.Sprintf("My Money: %.0f", money))
	moneyLabel.Refresh()
}

func updateNumOfPlayers(players int) {
	numOfPlayers.SetText(fmt.Sprintf("# of players: %d", players))
	numOfPlayers.Refresh()
}

func updateAddress(addresses []string) {
	loopbackAddress = addresses[0]
	lanAddress = addresses[1]
}

func updateCards(playerInfo channelmanager.PlayerInfo) {
	playerCards.Objects = nil

	for playerIndex, playerNickname := range playerInfo.Players {
		newCard := 	widget.NewCard(playerNickname, fmt.Sprintf("$%.0f", playerInfo.Money[playerIndex]), nil)
		playerCards.Add(newCard)
	}
	
	playerCards.Refresh()
}
