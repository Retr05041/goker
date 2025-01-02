package gui

import (
	"fmt"
	"fyne.io/fyne/v2/widget"
	"goker/internal/p2p"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2"
)

var (

	// showMenuUI
	inputedAddress = widget.NewEntry()
	banner = canvas.NewText("Goker", BLUE)
	hostOrConnect string
	submit *widget.Button
	peerType *widget.RadioGroup

	// showHostUI
	copyAddrButton *widget.Button
	testProtocolFS *widget.Button
	testProtocolSS *widget.Button
	DisplayDeck *widget.Button

	// showConnectedUI
	thankLabel = widget.NewLabel("Connected to Host!")

	// showGameScreen
	foldButton *widget.Button
	raiseButton *widget.Button
	callButton *widget.Button
	checkButton *widget.Button
)

func setElements() {
	// showMenuUI
	inputedAddress.SetPlaceHolder("Host address...")
	inputedAddress.Disable()

	banner.TextSize = 24
	banner.TextStyle = fyne.TextStyle{Bold: true, Italic: false}
	banner.Alignment = fyne.TextAlignCenter

	submit = widget.NewButton("Submit", func() {
		fmt.Println("Choice made: ", hostOrConnect)
		if hostOrConnect == "Connect" {
			go myself.Init(false, inputedAddress.Text)
			showConnectedUI(mainWindow)
		} else if hostOrConnect == "Host" {
			go myself.Init(true, "")
			showHostUI(mainWindow)
		}
	})
	submit.Disable()

	peerType = widget.NewRadioGroup([]string{"Host", "Connect"}, func(value string) {
		submit.Enable()
		if value == "Connect" {
			inputedAddress.Enable()
		} else {
			inputedAddress.Disable()
		}
		hostOrConnect = value
	})

	// showHostUI
	copyAddrButton = widget.NewButton("Copy server address", func() {
		mainWindow.Clipboard().SetContent(myself.ThisHostMultiaddr)
	})
	testProtocolFS = widget.NewButton("Test ProtocolFS", func() {
		myself.ExecuteCommand(&p2p.ProtocolFirstStep{})
		fmt.Println("Test ProtocolFS done on all peers.")
	})
	testProtocolSS = widget.NewButton("Test ProtocolSS", func() {
		myself.ExecuteCommand(&p2p.ProtocolSecondStep{})
		fmt.Println("Test ProtocolSS done on all peers.")
	})
	DisplayDeck = widget.NewButton("Display Current Deck", func() {
		myself.DisplayDeck()
		fmt.Println("Deck Displayed.")
	})

	// showGameScreen
	foldButton = widget.NewButton("Fold", func() {
		fmt.Println("Fold was pressed")
	})
	raiseButton = widget.NewButton("Raise", func() {
		fmt.Println("Raise was pressed")
	})
	callButton = widget.NewButton("Call", func() {
		fmt.Println("Call was pressed")
	})
	checkButton = widget.NewButton("Check", func() {
		fmt.Println("Check was pressed")
	})


}
