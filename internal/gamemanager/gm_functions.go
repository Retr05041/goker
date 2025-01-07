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

		gm.state.MyHand = append(gm.state.MyHand, cardImage)
	}

	channelmanager.HandChannel <- gm.state.MyHand // Send the new hand through the channel
}

// Setup board with back of cards
func (gm *GameManager) initBoard() {
	for i := 0; i < 5; i++ {
		cardImage := canvas.NewImageFromFile("media/svg_playing_cards/backs/png_96_dpi/red.png")
		cardImage.FillMode = canvas.ImageFillOriginal

		gm.state.Board = append(gm.state.Board, cardImage)
	}

	channelmanager.BoardChannel <- gm.state.Board
}

// Load two cards into your hand and update the grid
func (gm *GameManager) loadHand(cardOneName, cardTwoName string) {
	cardOne := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardOneName + ".png")
	cardOne.FillMode = canvas.ImageFillOriginal

	cardTwo := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardTwoName + ".png")
	cardTwo.FillMode = canvas.ImageFillOriginal

	gm.state.MyHand[0] = cardOne
	gm.state.MyHand[1] = cardTwo
	channelmanager.HandChannel <- gm.state.MyHand
}

func (gm *GameManager) loadFlop(cardOneName, cardTwoName, cardThreeName string) {
	cardOne := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardOneName + ".png")
	cardOne.FillMode = canvas.ImageFillOriginal

	cardTwo := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardTwoName + ".png")
	cardTwo.FillMode = canvas.ImageFillOriginal

	cardThree := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardThreeName + ".png")
	cardThree.FillMode = canvas.ImageFillOriginal

	gm.state.Board[0] = cardOne
	gm.state.Board[1] = cardTwo
	gm.state.Board[2] = cardThree
	channelmanager.BoardChannel <- gm.state.Board
}

func (gm *GameManager) loadTurn(cardName string) {
	card := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardName + ".png")
	card.FillMode = canvas.ImageFillOriginal

	gm.state.Board[3] = card
	channelmanager.BoardChannel <- gm.state.Board
}

func (gm *GameManager) loadRiver(cardName string) {
	card := canvas.NewImageFromFile("media/svg_playing_cards/fronts/png_96_dpi/" + cardName + ".png")
	card.FillMode = canvas.ImageFillOriginal

	gm.state.Board[4] = card
	channelmanager.BoardChannel <- gm.state.Board
}
