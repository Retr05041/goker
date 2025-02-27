package gui

import (
	"fmt"
	"goker/internal/channelmanager"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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
	go gmListener(mainWindow)

	// Init everything on the GM side
	channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "Init"}

	// Run the first scene
	showMenuUI(mainWindow)

	// Run GUI
	mainWindow.Resize(fyne.NewSize(float32(MAX_WIDTH), float32(MAX_HEIGHT))) // Set the window size
	mainWindow.ShowAndRun()
}

// Since we want to keep logic out of the GUI, this will listen for any updates from specific parts of the state
func gmListener(window fyne.Window) {
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
		case address := <-channelmanager.TGUI_AddressChan:
			updateAddress(address)
		case playerInfo := <-channelmanager.TGUI_PlayerInfo:
			window.SetTitle("Goker - " + playerInfo.Me)
			updateCards(playerInfo)
		case <-channelmanager.TGUI_StartRound:
			showGameScreen(window)
		case <-channelmanager.TGUI_EndRound:
			if isHost {
				showHostUI(window)
			} else {
				showConnectedUI(window)
			}
		}
	}
}

func updateHandImages(hand []*canvas.Image) {
	newGrid := container.NewGridWithColumns(2)
	for _, image := range hand {
		newGrid.Add(image)
	}
	handGrid.Objects = []fyne.CanvasObject{newGrid}
	handGrid.Refresh()
}

func updateBoardImages(board []*canvas.Image) {
	newGrid := container.NewGridWithColumns(5)
	for _, image := range board {
		newGrid.Add(image)
	}
	boardGrid.Objects = []fyne.CanvasObject{newGrid}
	boardGrid.Refresh()
}

func updatePot(pot float64) {
	potLabel.SetText(fmt.Sprintf("Pot: %.0f", pot))
	potLabel.Refresh()
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

	highestBet = playerInfo.HighestBet
	myBetsSoFar = playerInfo.MyBetSoFar

	if playerInfo.Me == playerInfo.WhosTurn {
		foldButton.Enable()
		raiseButton.Enable()
		callButton.Enable()
		checkButton.Enable()
	} else {
		foldButton.Disable()
		raiseButton.Disable()
		callButton.Disable()
		checkButton.Disable()
	}

	for playerIndex, playerNickname := range playerInfo.Players {
		if playerNickname == playerInfo.Me {
			myMoney = playerInfo.Money[playerIndex]
		}

		var titleText *canvas.Text
		if playerNickname == playerInfo.WhosTurn {
			titleText = canvas.NewText(playerNickname, BLUE)

		} else {
			titleText = canvas.NewText(playerNickname, color.White)
		}
		titleText.TextSize = 18
		titleText.TextStyle.Bold = true

		moneyText := canvas.NewText(fmt.Sprintf("$%.0f", playerInfo.Money[playerIndex]), color.White)
		moneyText.TextSize = 16
		moneyText.TextStyle.Bold = true

		card := container.NewVBox(
			titleText,
			moneyText,
		)

		playerCards.Add(card)
	}
	playerCards.Refresh()
}
