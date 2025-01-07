package gui

import (
	"fmt"

	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/container"
)

var (
	handGrid    = container.NewGridWithColumns(2) // Holds the hand images
	boardGrid   = container.NewGridWithColumns(5) // Holds the board iamges

	moneyLabel = widget.NewLabel(fmt.Sprintf("My Money: %.0f", 0.0))
	valueLabel = widget.NewLabel(fmt.Sprintf("Value: %.0f", 0.0))
	betSlider = widget.NewSlider(0, 0)
	potLabel = widget.NewLabel(fmt.Sprintf("Pot: %.0f", 0.0))
)

func initElements() {
	betSlider.Step = 1
	betSlider.OnChanged = func(f float64) {
		valueLabel.SetText(fmt.Sprintf("Value: %.0f", f))
	}
}
