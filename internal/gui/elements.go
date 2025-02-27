package gui

import (
	"fmt"
	"goker/internal/channelmanager"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var (
	// Lobby
	numOfPlayers    = widget.NewLabel(fmt.Sprintf("# of players: %d", 1))
	loopbackAddress string
	lanAddress      string
	isHost          bool

	// Game
	boardSize   = fyne.NewSize((234*5)/2, 333/2)   // 234x333 per card
	handSize    = fyne.NewSize((234*2)/2, 333/2)   // 234x333
	handGrid    = container.NewGridWrap(handSize)  // Holds the hand images
	boardGrid   = container.NewGridWrap(boardSize) // Holds the hand images
	foldButton  *widget.Button
	raiseButton *widget.Button
	callButton  *widget.Button
	checkButton *widget.Button

	playerCards = container.NewVBox()

	myMoney     = 0.0
	highestBet  = 0.0
	myBetsSoFar = 0.0
	valueLabel  = widget.NewLabel(fmt.Sprintf("$%.0f", 0.0))
	betSlider   = widget.NewSlider(0, 100)
	potLabel    = widget.NewLabel(fmt.Sprintf("Pot: $%.0f", 0.0))
)

func initElements() {
	betSlider.Step = 1
	betSlider.OnChanged = func(f float64) {
		valueLabel.SetText(fmt.Sprintf("$%.0f", f))
	}

	foldButton = widget.NewButton("Fold", func() {
		channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "Fold"}
	})
	raiseButton = widget.NewButton("Raise", func() {
		if (betSlider.Value <= myMoney) && (betSlider.Value+myBetsSoFar > highestBet) {
			channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "Raise", DataF: betSlider.Value}
		}
	})
	callButton = widget.NewButton("Call", func() {
		if highestBet <= myMoney {
			channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "Call"}
		}
	})
	checkButton = widget.NewButton("Check", func() {
		channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "Check"}
	})
}
