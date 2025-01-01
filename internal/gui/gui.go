package gui

import (
	"fmt"
	"goker/internal/p2p"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func Init() {
	// Setup myself
	myself = new(p2p.GokerPeer)
	// Setup hand
	myHand = make(map[string]*canvas.Image, 2)
	// Setup GUI
	myApp := app.New()
	myWindow := myApp.NewWindow("Goker")

	// Setup Main Menu
	showGameScreen(myWindow)

	// Run GUI
	myWindow.Resize(fyne.NewSize(float32(MAX_WIDTH), float32(MAX_HEIGHT))) // Set the window size
	myWindow.ShowAndRun()
}

func showMenuUI(myWindow fyne.Window) {
	var choice string

	banner := canvas.NewText("Goker", BLUE)
	banner.TextSize = 24
	banner.TextStyle = fyne.TextStyle{Bold: true, Italic: false}
	banner.Alignment = fyne.TextAlignCenter

	inputedAddress := widget.NewEntry()
	inputedAddress.SetPlaceHolder("Host address...")
	inputedAddress.Disable()

	submit := widget.NewButton("Submit", func() {
		fmt.Println("Choice made: ", choice)
		if choice == "Connect" {
			go myself.Init(false, inputedAddress.Text)
			showConnectedUI(myWindow)
		} else if choice == "Host" {
			go myself.Init(true, "")
			showHostUI(myWindow)
		}
	})
	submit.Disable()

	peerType := widget.NewRadioGroup([]string{"Host", "Connect"}, func(value string) {
		submit.Enable()
		if value == "Connect" {
			inputedAddress.Enable()
		} else {
			inputedAddress.Disable()
		}
		choice = value
	})

	menuContent := container.NewGridWrap(fyne.NewSize(float32(MAX_WIDTH), float32(MAX_HEIGHT)), container.NewVBox(banner, peerType, inputedAddress, submit))
	myWindow.SetContent(container.NewCenter(menuContent))
}

// What the host will see
func showHostUI(myWindow fyne.Window) {
	copyAddrButton := widget.NewButton("Copy server address", func() {
		myWindow.Clipboard().SetContent(myself.ThisHostMultiaddr)
	})
	testProtocolFS := widget.NewButton("Test ProtocolFS", func() {
		myself.ExecuteCommand(&p2p.ProtocolFirstStep{})
		fmt.Println("Test ProtocolFS done on all peers.")
	})
	testProtocolSS := widget.NewButton("Test ProtocolSS", func() {
		myself.ExecuteCommand(&p2p.ProtocolSecondStep{})
		fmt.Println("Test ProtocolSS done on all peers.")
	})
	DisplayDeck := widget.NewButton("Display Current Deck", func() {
		myself.DisplayDeck()
		fmt.Println("Deck Displayed.")
	})
	myWindow.SetContent(container.NewVBox(copyAddrButton, testProtocolFS, testProtocolSS, DisplayDeck))
}

// What a connected peer will see
func showConnectedUI(myWindow fyne.Window) {
	thankLabel := widget.NewLabel("Connected to Host!")
	DisplayDeck := widget.NewButton("Display Current Deck", func() {
		myself.DisplayDeck()
		fmt.Println("Deck Displayed.")
	})
	myWindow.SetContent(container.NewVBox(thankLabel, DisplayDeck))
}

func showGameScreen(givenWindow fyne.Window) {
	// Call setupCards to initialize the playingCards map
	loadCard("spades_ace")
	loadCard("hearts_ace")

	// Create a container to hold all the card images in a grid
	grid := container.NewGridWithColumns(2) // Adjust the number of columns as desired

	// Add each card image to the grid
	for _, cardImage := range myHand {
		grid.Add(cardImage)
	}

	foldButton := widget.NewButton("Fold", func() {
		fmt.Println("Fold was pressed")
	})
	raiseButton := widget.NewButton("Raise", func() {
		fmt.Println("Raise was pressed")
	})
	callButton := widget.NewButton("Call", func() {
		fmt.Println("Call was pressed")
	})
	checkButton := widget.NewButton("Check", func() {
		fmt.Println("Check was pressed")
	})

	// Center the grid in the window
	givenWindow.SetContent(container.NewCenter(container.NewHBox(grid, container.NewVBox(foldButton, raiseButton, callButton, checkButton))))
}
