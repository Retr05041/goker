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

	myMoney            = 0.0
	highestBet         = 0.0
	myBetsForThisPhase = 0.0
	valueLabel         = widget.NewLabel(fmt.Sprintf("$%.0f", 0.0))
	betSlider          = widget.NewSlider(0, 100)
	potLabel           = widget.NewLabel(fmt.Sprintf("Pot: $%.0f", 0.0))
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
		if (betSlider.Value <= myMoney) && ((betSlider.Value+myBetsForThisPhase > highestBet) || (highestBet == 0)) {
			channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "Raise", DataF: betSlider.Value}
		}
	})
	callButton = widget.NewButton("Call", func() {
		if (highestBet-myBetsForThisPhase <= myMoney) && (highestBet != 0) { // If the current highest bet is less than my money, we can call
			highestBet = 0 // In case no one raises after us, we obv don't want to be able to call again
			channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "Call"}
		}
	})
	checkButton = widget.NewButton("Check", func() {
		// Just check.. however I will need to make sure no ones raised yet
		if highestBet == 0 {
			channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "Check"}
		}
	})
}
