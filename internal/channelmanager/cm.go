package channelmanager

import (
	"goker/internal/gamestate"

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

	// Channles for network (<- Network)
	FNET_NumOfPlayersChan chan int
	FNET_StartRoundChan   chan bool

	// Game state to be sent from game manager to network - Singular channels will be used to communicate with GUI
	TFNET_GameStateChan chan gamestate.GameState // We don't make this a pointer so we can work with copies
)

// Actions made by the user on the GUI
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

	FNET_NumOfPlayersChan = make(chan int)
	FNET_StartRoundChan = make(chan bool)

	TFNET_GameStateChan = make(chan gamestate.GameState)
}
