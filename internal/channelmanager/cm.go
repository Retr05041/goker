package channelmanager

import (
	"fyne.io/fyne/v2/canvas"
)

var (
	// Channels for user input (<- GUI)
	FGUI_InitChan   chan bool
	FGUI_ActionChan chan ActionType

	// Channels for specific elements in the UI (-> GUI)
	TGUI_AddressChan chan []string        // Host address - Set during init
	TGUI_HandChan    chan []*canvas.Image // Current hand images
	TGUI_BoardChan   chan []*canvas.Image // Current board images

	TGUI_PotChan    chan float64 // Pot
	TGUI_PlayerInfo chan PlayerInfo
	TGUI_StartRound chan struct{} // For telling the GUI to start the round

	// Channles for network (<- Network)
	FNET_NetActionDoneChan chan struct{}
	FNET_NumOfPlayersChan  chan int
	FNET_StartRoundChan    chan bool

	// Game state to be sent from game manager to network - Singular channels will be used to communicate with GUI
	TNET_ActionChan chan ActionType
)

// Actions made by the user on the GUI
type ActionType struct {
	Action string

	// Possible data needed for an action
	DataF float64
	DataS []string
}

// Player info for the GUI to use - sent from the game manager
type PlayerInfo struct {
	Players []string
	Money   []float64
	Me      string
}

// Initialize all channels
func Init() {
	FGUI_InitChan = make(chan bool)
	FGUI_ActionChan = make(chan ActionType)

	TGUI_AddressChan = make(chan []string)
	TGUI_HandChan = make(chan []*canvas.Image)
	TGUI_BoardChan = make(chan []*canvas.Image)

	TGUI_PotChan = make(chan float64)
	TGUI_PlayerInfo = make(chan PlayerInfo)
	TGUI_StartRound = make(chan struct{})

	FNET_NetActionDoneChan = make(chan struct{})
	FNET_NumOfPlayersChan = make(chan int)
	FNET_StartRoundChan = make(chan bool)

	TNET_ActionChan = make(chan ActionType)
}
