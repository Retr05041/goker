package channelmanager

import (
	"fyne.io/fyne/v2/canvas"
)

var (
	// Channels for user input (<- GUI)
	InitHandAndBoard   chan bool
	HostConnectChannel chan string
	ActionChannel      chan ActionType

	// Channels for specific elements in the UI (-> GUI)
	HandChannel    chan []*canvas.Image
	BoardChannel   chan []*canvas.Image
	PotChannel     chan float64
	MyMoneyChannel chan float64
)

type ActionType struct {
	Action string
	Data   *float64
}

// Initialize all channels
func Init() {
	InitHandAndBoard = make(chan bool)
	HostConnectChannel = make(chan string)
	ActionChannel = make(chan ActionType)

	HandChannel = make(chan []*canvas.Image)
	BoardChannel = make(chan []*canvas.Image)
	PotChannel = make(chan float64)
	MyMoneyChannel = make(chan float64)
}
