package gui

import (
	"image/color"
	"goker/internal/p2p"
)

var (
	// Myself
	myself *p2p.GokerPeer

	// Window Settings
	MAX_WIDTH = 300
	MAX_HEIGHT = 200

	// Colors
	BLUE = color.NRGBA{R: 0, G: 173, B: 216, A: 255}

)
