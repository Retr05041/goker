package gui

import (
	"goker/internal/p2p"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

var (
	// Info about the user
	myself *p2p.GokerPeer
	myMoney float64

	// Table Info
	pot float64
	minBet float64

	// Card Variables
	myHand []*canvas.Image // Images for the cards in my hand
	theBoard []*canvas.Image // Images for the cards in the board
	handGrid = container.NewGridWithColumns(2) // Holds the hand images
	boardGrid = container.NewGridWithColumns(5) // Holds the board iamges

	ranks = [...]string{"ace", "2", "3", "4", "5", "6", "7", "8", "9", "10", "jack", "queen", "king"}
	suits = [...]string{"hearts", "diamonds", "clubs", "spades"}

	// GUI Settings
	MAX_WIDTH  = 600
	MAX_HEIGHT = 400

	// Colors
	BLUE = color.NRGBA{R: 0, G: 173, B: 216, A: 255}
)

// Runner for gui
func Init() {
	// Setup myself
	myself = new(p2p.GokerPeer)
	myMoney = 100.0
	minBet = 5.0
	pot = 0.0

	// Setup GUI
	myApp := app.New()
	mainWindow := myApp.NewWindow("Goker")

	// Setup hand
	initHand()
	initBoard()

	// Make sure scenes are prepped
	showGameScreen(mainWindow)

	// Run GUI
	mainWindow.Resize(fyne.NewSize(float32(MAX_WIDTH), float32(MAX_HEIGHT))) // Set the window size
	mainWindow.ShowAndRun()
}

// ### SIMPLE HAND MANIPULATION FUNCTIONS ###

// Setup hand with back of cards
func initHand() {
	for i := 0; i < 2; i++ {
		cardImage := canvas.NewImageFromFile("media/svg_playing_cards/backs/png_96_dpi/red.png")
		cardImage.FillMode = canvas.ImageFillOriginal

		myHand = append(myHand, cardImage)
	}

	for _, image := range myHand {
		handGrid.Add(image)
	}
}

// Setup board with back of cards
func initBoard() {
	for i := 0; i < 5; i++ {
		cardImage := canvas.NewImageFromFile("media/svg_playing_cards/backs/png_96_dpi/red.png")
		cardImage.FillMode = canvas.ImageFillOriginal

		theBoard = append(theBoard, cardImage)
	}

	for _, image := range theBoard {
		boardGrid.Add(image)
	}
}

// Load two cards into your hand and update the grid
func loadHand(cardOneName, cardTwoName string) {
	cardOne := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardOneName + ".png")
	cardOne.FillMode = canvas.ImageFillOriginal

	cardTwo := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardTwoName + ".png")
	cardTwo.FillMode = canvas.ImageFillOriginal

	myHand[0] = cardOne
	myHand[1] = cardTwo

	handGrid.Objects = nil
	for _, image := range myHand {
		handGrid.Add(image)
	}
	handGrid.Refresh()
}


func loadFlop(cardOneName, cardTwoName, cardThreeName string) {
	cardOne := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardOneName + ".png")
	cardOne.FillMode = canvas.ImageFillOriginal

	cardTwo := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardTwoName + ".png")
	cardTwo.FillMode = canvas.ImageFillOriginal

	cardThree := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardThreeName + ".png")
	cardThree.FillMode = canvas.ImageFillOriginal

	theBoard[0] = cardOne
	theBoard[1] = cardTwo
	theBoard[2] = cardThree

	boardGrid.Objects = nil
	for _, image := range theBoard {
		boardGrid.Add(image)
	}
	boardGrid.Refresh()
}

func loadTurn(cardName string) {
	card := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardName + ".png")
	card.FillMode = canvas.ImageFillOriginal

	theBoard[3] = card

	boardGrid.Objects = nil
	for _, image := range theBoard {
		boardGrid.Add(image)
	}
	boardGrid.Refresh()
}

func loadRiver(cardName string) {
	card := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardName + ".png")
	card.FillMode = canvas.ImageFillOriginal

	theBoard[4] = card

	boardGrid.Objects = nil
	for _, image := range theBoard {
		boardGrid.Add(image)
	}
	boardGrid.Refresh()
}
