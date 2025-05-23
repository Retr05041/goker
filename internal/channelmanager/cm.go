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

	TGUI_PotChan         chan float64 // Pot
	TGUI_PlayerInfo      chan PlayerInfo
	TGUI_StartRound      chan struct{} // For telling the GUI to start the round
	TGUI_EndRound        chan struct{} // For telling the GUI to start the round
	TGUI_ShowLoadingChan chan struct{} // Show the loading screen
	TGUI_MoveToLobby     chan bool     // Move to lobby, bool is if host or not

	// Channles for network (<- Network)
	FNET_NetActionDoneChan chan struct{}
	FNET_NumOfPlayersChan  chan int
	FNET_StartRoundChan    chan bool

	// Game state to be sent from game manager to network - Singular channels will be used to communicate with GUI
	TNET_ActionChan chan ActionType

	TGM_PhaseCheck      chan struct{} // Used when switching turns, will make gm check if there is a phase shift needed
	TGM_EndRound        chan struct{} // For state telling the GM that this round is over and to reset and move to next round
	TGS_PhaseSwitchDone chan struct{} // For the GM to tell the GS to continue with the "Next Turn" as the phase has been switched
	TGM_WaitForPuzzles  chan struct{}
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
	Players            []string
	Money              []float64
	Me                 string
	HighestBet         float64 // Highest bet by the users so far
	WhosTurn           string  // the nickname
	MyBetsForThisPhase float64 // What I have bet so far
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
	TGUI_EndRound = make(chan struct{})
	TGUI_ShowLoadingChan = make(chan struct{})
	TGUI_MoveToLobby = make(chan bool)

	FNET_NetActionDoneChan = make(chan struct{})
	FNET_NumOfPlayersChan = make(chan int)
	FNET_StartRoundChan = make(chan bool)

	TNET_ActionChan = make(chan ActionType)

	TGM_PhaseCheck = make(chan struct{})
	TGM_EndRound = make(chan struct{})
	TGS_PhaseSwitchDone = make(chan struct{})
	TGM_WaitForPuzzles = make(chan struct{})
}
