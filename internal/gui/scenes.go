package gui

import (
	"fmt"

	"goker/internal/p2p"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/canvas"
)

func showMenuUI(givenWindow fyne.Window) {
	var hostOrConnect string

	inputedAddress := widget.NewEntry()
	inputedAddress.SetPlaceHolder("Host address...")
	inputedAddress.Disable()


	banner := canvas.NewText("Goker", BLUE)
	banner.TextSize = 24
	banner.TextStyle = fyne.TextStyle{Bold: true, Italic: false}
	banner.Alignment = fyne.TextAlignCenter

	submit := widget.NewButton("Submit", func() {
		fmt.Println("Choice made: ", hostOrConnect)
		if hostOrConnect == "Connect" {
			go myself.Init(false, inputedAddress.Text)
			showConnectedUI(givenWindow)
		} else if hostOrConnect == "Host" {
			go myself.Init(true, "")
			showHostUI(givenWindow)
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
		hostOrConnect = value
	})


	
	givenWindow.SetContent(
		container.NewCenter(
			container.NewGridWrap(
				fyne.NewSize(float32(MAX_WIDTH), float32(MAX_HEIGHT)), 
				container.NewVBox(banner, peerType, inputedAddress, submit))))
}

// What the host will see
func showHostUI(givenWindow fyne.Window) {
	copyAddrButton := widget.NewButton("Copy server address", func() {
		givenWindow.Clipboard().SetContent(myself.ThisHostMultiaddr)
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

	givenWindow.SetContent(
		container.NewVBox(copyAddrButton, testProtocolFS, testProtocolSS, DisplayDeck))
}

// What a connected peer will see
func showConnectedUI(myWindow fyne.Window) {
	thankLabel := widget.NewLabel("Connected to Host!")

	myWindow.SetContent(
		container.NewVBox(thankLabel))
}

func showGameScreen(givenWindow fyne.Window) {
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

	// Call setupCards to initialize the playingCards map
	loadCard("spades_ace")
	loadCard("hearts_ace")

	// Create a container to hold all the card images in a grid
	grid := container.NewGridWithColumns(2) // Adjust the number of columns as desired

	// Add each card image to the grid
	for _, cardImage := range myHand {
		grid.Add(cardImage)
	}

	// Center the grid in the window
	givenWindow.SetContent(
		container.NewCenter(
			container.NewHBox(grid, container.NewVBox(foldButton, raiseButton, callButton, checkButton))))
}
