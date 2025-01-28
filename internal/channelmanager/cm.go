package channelmanager

import (
	"fyne.io/fyne/v2/canvas"
)

var (
	// Channels for user input (<- GUI)
	FGUI_InitChan   chan bool
	FGUI_ActionChan chan ActionType

	// Channels for specific elements in the UI (-> GUI)
	TGUI_HandChan    chan []*canvas.Image
	TGUI_BoardChan   chan []*canvas.Image
	TGUI_PotChan     chan float64
	TGUI_MyMoneyChan chan float64
	TGUI_AddressChan chan []string

	// Channles for network
	FNET_InitDoneChan        chan struct{}
	FNET_NumOfPlayersChan    chan int
	FNET_StartRoundChan chan bool
)

type ActionType struct {
	Action string
	DataF  *float64
	DataS  *string
}

// Initialize all channels
func Init() {
	FGUI_InitChan = make(chan bool)
	FGUI_ActionChan = make(chan ActionType)

	TGUI_HandChan = make(chan []*canvas.Image)
	TGUI_BoardChan = make(chan []*canvas.Image)
	TGUI_PotChan = make(chan float64)
	TGUI_MyMoneyChan = make(chan float64)
	TGUI_AddressChan = make(chan []string)

	FNET_InitDoneChan = make(chan struct{})
	FNET_NumOfPlayersChan = make(chan int)
	FNET_StartRoundChan = make(chan bool)
}
