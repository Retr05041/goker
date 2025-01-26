package channelmanager

import (
	"fyne.io/fyne/v2/canvas"
)

var (
	// Channels for user input (<- GUI)
	InitHandAndBoard   chan bool
	ActionChannel      chan ActionType

	// Channels for specific elements in the UI (-> GUI)
	HandChannel    chan []*canvas.Image
	BoardChannel   chan []*canvas.Image
	PotChannel     chan float64
	MyMoneyChannel chan float64
	PlayersChannel chan int
	AddressChannel chan []string

	// Channles for network
	NetworkInitDoneChannel chan struct{}
)

type ActionType struct {
	Action string
	DataF  *float64
	DataS  *string
}

// Initialize all channels
func Init() {
	InitHandAndBoard = make(chan bool)
	ActionChannel = make(chan ActionType)

	HandChannel = make(chan []*canvas.Image)
	BoardChannel = make(chan []*canvas.Image)
	PotChannel = make(chan float64)
	MyMoneyChannel = make(chan float64)
	PlayersChannel = make(chan int)
	AddressChannel = make(chan []string)

	NetworkInitDoneChannel = make(chan struct{})
}
