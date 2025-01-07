package gui

import (
	"fmt"
	"goker/internal/channelmanager"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
)

var (
	// GUI Settings
	MAX_WIDTH  = 600
	MAX_HEIGHT = 400

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
	channelmanager.ActionChannel <- channelmanager.ActionType{
		Action: "Init",
		Data: nil,
	}

	// Run the first scene
	showGameScreen(mainWindow)

	// Run GUI
	mainWindow.Resize(fyne.NewSize(float32(MAX_WIDTH), float32(MAX_HEIGHT))) // Set the window size
	mainWindow.ShowAndRun()
}

func gmListener() {
	for {
		select {
		case hand := <-channelmanager.HandChannel: // Hand = whatever is coming in from the handChannel
			updateHandImages(hand)
		case board := <-channelmanager.BoardChannel:
			updateBoardImages(board)
		case pot := <-channelmanager.PotChannel:
			updatePot(pot)
		case money := <-channelmanager.MyMoneyChannel:
			updateMyMoney(money)
		}
	}
}

func updateHandImages(hand []*canvas.Image) {
	fmt.Println("Updating Hand Images")
	handGrid.Objects = nil
	for _, image := range hand {
		handGrid.Add(image)
	}
	handGrid.Refresh()
}

func updateBoardImages(board []*canvas.Image) {
	fmt.Println("Updating Board Images")
	boardGrid.Objects = nil
	for _, image := range board {
		boardGrid.Add(image)
	}
	boardGrid.Refresh()
}

func updatePot(pot float64) {
	potLabel.SetText(fmt.Sprintf("Pot: %.0f", pot))
	potLabel.Refresh()
}

func updateMyMoney(money float64) {
	moneyLabel.SetText(fmt.Sprintf("My Money: %.0f", money))
	moneyLabel.Refresh()
}
