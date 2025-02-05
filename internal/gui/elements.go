package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var (
	// Lobby
	numOfPlayers = widget.NewLabel(fmt.Sprintf("# of players: %d", 1))
	loopbackAddress string
	lanAddress string

	// Game
	boardSize = fyne.NewSize((234*5)/2, 333/2) // 234x333 per card
	handSize = fyne.NewSize((234*2)/2, 333/2) // 234x333
	handGrid  = container.NewGridWrap(boardSize) // Holds the hand images
	boardGrid = container.NewGridWrap(boardSize) // Holds the hand images

	moneyLabel = widget.NewLabel(fmt.Sprintf("My Money: $%.0f", 0.0))
	valueLabel = widget.NewLabel(fmt.Sprintf("$%.0f", 0.0))
	betSlider  = widget.NewSlider(0, 100)
	potLabel   = widget.NewLabel(fmt.Sprintf("Pot: $%.0f", 0.0))
)

func initElements() {
	betSlider.Step = 1
	betSlider.OnChanged = func(f float64) {
		valueLabel.SetText(fmt.Sprintf("$%.0f", f))
	}
}
