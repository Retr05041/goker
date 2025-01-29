package gamemanager

import (
	"fyne.io/fyne/v2/canvas"
	"goker/internal/channelmanager"
)

// Setup hand with back of cards
func (gm *GameManager) initHand() {
	for i := 0; i < 2; i++ {
		cardImage := canvas.NewImageFromFile("media/svg_playing_cards/backs/png_96_dpi/red.png")
		cardImage.FillMode = canvas.ImageFillOriginal

		gm.MyHand = append(gm.MyHand, cardImage)
	}

	channelmanager.TGUI_HandChan <- gm.MyHand // Send the new hand through the channel
}

// Setup board with back of cards
func (gm *GameManager) initBoard() {
	for i := 0; i < 5; i++ {
		cardImage := canvas.NewImageFromFile("media/svg_playing_cards/backs/png_96_dpi/red.png")
		cardImage.FillMode = canvas.ImageFillOriginal

		gm.Board = append(gm.Board, cardImage)
	}

	channelmanager.TGUI_BoardChan <- gm.Board
}

// Load two cards into your hand and update the grid
func (gm *GameManager) loadHand(cardOneName, cardTwoName string) {
	cardOne := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardOneName + ".png")
	cardOne.FillMode = canvas.ImageFillOriginal

	cardTwo := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardTwoName + ".png")
	cardTwo.FillMode = canvas.ImageFillOriginal

	gm.MyHand[0] = cardOne
	gm.MyHand[1] = cardTwo
	channelmanager.TGUI_HandChan <- gm.MyHand
}

func (gm *GameManager) loadFlop(cardOneName, cardTwoName, cardThreeName string) {
	cardOne := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardOneName + ".png")
	cardOne.FillMode = canvas.ImageFillOriginal

	cardTwo := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardTwoName + ".png")
	cardTwo.FillMode = canvas.ImageFillOriginal

	cardThree := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardThreeName + ".png")
	cardThree.FillMode = canvas.ImageFillOriginal

	gm.Board[0] = cardOne
	gm.Board[1] = cardTwo
	gm.Board[2] = cardThree
	channelmanager.TGUI_BoardChan <- gm.Board
}

func (gm *GameManager) loadTurn(cardName string) {
	card := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardName + ".png")
	card.FillMode = canvas.ImageFillOriginal

	gm.Board[3] = card
	channelmanager.TGUI_BoardChan <- gm.Board
}

func (gm *GameManager) loadRiver(cardName string) {
	card := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardName + ".png")
	card.FillMode = canvas.ImageFillOriginal

	gm.Board[4] = card
	channelmanager.TGUI_BoardChan <- gm.Board
}
