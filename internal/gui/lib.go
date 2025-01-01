package gui

import (
	"fmt"
	"goker/internal/p2p"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

var (
	// Myself
	myself *p2p.GokerPeer

	myHand map[string]*canvas.Image
	ranks  = [...]string{"ace", "2", "3", "4", "5", "6", "7", "8", "9", "10", "jack", "queen", "king"}
	suits  = [...]string{"hearts", "diamonds", "clubs", "spades"}

	// Window Settings
	MAX_WIDTH  = 300
	MAX_HEIGHT = 200

	// Colors
	BLUE = color.NRGBA{R: 0, G: 173, B: 216, A: 255}
)

func loadCard(cardName string) {
	image := canvas.NewImageFromFile(fmt.Sprintf("media/svg_playing_cards/fronts/" + cardName + ".svg"))
	image.FillMode = canvas.ImageFillOriginal
	image.Resize(fyne.NewSize(image.Size().Width/2, image.Size().Height/2))

	myHand[cardName] = image
}
