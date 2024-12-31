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

	// Setup GUI
	myApp := app.New()
	myWindow := myApp.NewWindow("Goker")

	// Setup Main Menu
	showCardUI(myWindow)

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

func showCardUI(givenWindow fyne.Window) {
	image := canvas.NewImageFromFile("media/svg_playing_cards/fronts/spades_ace.svg")
	image.FillMode = canvas.ImageFillOriginal
	image.Resize(fyne.NewSize(image.Size().Width/2, image.Size().Height/2))

	// Set the image to be the window's content
	container := container.NewCenter(image)
	givenWindow.SetContent(container)
}
