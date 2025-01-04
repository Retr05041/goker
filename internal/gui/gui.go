package gui

import (
	"goker/internal/p2p"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
)

var (
	// Myself
	myself *p2p.GokerPeer

	myHand []*canvas.Image
	theBoard []*canvas.Image
	ranks = [...]string{"ace", "2", "3", "4", "5", "6", "7", "8", "9", "10", "jack", "queen", "king"}
	suits = [...]string{"hearts", "diamonds", "clubs", "spades"}

	// Window Settings
	MAX_WIDTH  = 600
	MAX_HEIGHT = 400

	// Colors
	BLUE = color.NRGBA{R: 0, G: 173, B: 216, A: 255}
)

// Runner for gui
func Init() {
	// Setup myself
	myself = new(p2p.GokerPeer)

	// Setup hand
	initHand()
	initBoard()

	// Setup GUI
	myApp := app.New()
	mainWindow := myApp.NewWindow("Goker")

	// Make sure scenes are prepped
	showGameScreen(mainWindow)

	// Run GUI
	mainWindow.Resize(fyne.NewSize(float32(MAX_WIDTH), float32(MAX_HEIGHT))) // Set the window size
	mainWindow.ShowAndRun()
}

// ### SIMPLE HAND MANIPULATION FUNCTIONS ###
// used for inputing images into the hand and board (5 face up cards)

func loadHand(cardOneName, cardTwoName string) {
	cardOne := canvas.NewImageFromFile("media/svg_playing_cards/fronts/" + cardOneName + ".svg")
	cardOne.FillMode = canvas.ImageFillOriginal
	cardOne.Resize(fyne.NewSize(cardOne.Size().Width/2, cardOne.Size().Height/2))

	cardTwo := canvas.NewImageFromFile("media/svg_playing_cards/fronts/" + cardTwoName + ".svg")
	cardTwo.FillMode = canvas.ImageFillOriginal
	cardTwo.Resize(fyne.NewSize(cardOne.Size().Width/2, cardOne.Size().Height/2))

	myHand[0] = cardOne
	myHand[1] = cardTwo
}

func initHand() {
	for i := 0; i < 2; i++ {
		cardImage := canvas.NewImageFromFile("media/svg_playing_cards/backs/red.svg")
		cardImage.FillMode = canvas.ImageFillOriginal
		cardImage.Resize(fyne.NewSize(cardImage.Size().Width/2, cardImage.Size().Height/2))

		myHand = append(myHand, cardImage)
	}
}

func initBoard() {
	for i := 0; i < 5; i++ {
		cardImage := canvas.NewImageFromFile("media/svg_playing_cards/backs/red.svg")
		cardImage.FillMode = canvas.ImageFillOriginal
		cardImage.Resize(fyne.NewSize(cardImage.Size().Width/2, cardImage.Size().Height/2))

		theBoard = append(theBoard, cardImage)
	}
}
