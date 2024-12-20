package gui

import (
	"fmt"
	"goker/internal/p2p"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var myself *p2p.GokerPeer

func Init() {
	myself = new(p2p.GokerPeer) // Need to initialise myself so I can use it

	myApp := app.New()
	myWindow := myApp.NewWindow("Choice Widgets")

	var choice string

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

	menuContent := container.NewGridWrap(fyne.NewSize(300, 200), container.NewVBox(peerType, inputedAddress, submit))
	myWindow.SetContent(container.NewCenter(menuContent))
	myWindow.Show()
	myApp.Run()
	tidyUp()
}

func showHostUI(myWindow fyne.Window) {
	copyAddrButton := widget.NewButton("Copy server address", func() {
		myWindow.Clipboard().SetContent(myself.ThisHostMultiaddr)
	})
	pingButton := widget.NewButton("Ping All Peers", func() {
		myself.ExecuteCommand(&p2p.PingCommand{})
		fmt.Println("Ping command sent to all peers.")
	})
	testEncryptionButton := widget.NewButton("Test Commutative Encryption", func() {
		myself.ExecuteCommand(&p2p.TestEncryptionCommand{Message: "Hello, World."})
		fmt.Println("Testing encryption done on all peers.")
	})
	myWindow.SetContent(container.NewVBox(copyAddrButton, pingButton, testEncryptionButton))
}

func showConnectedUI(myWindow fyne.Window) {
	thankLabel := widget.NewLabel("Connected to Host!")
	pingButton := widget.NewButton("Ping All Peers", func() {
		myself.ExecuteCommand(&p2p.PingCommand{})
		fmt.Println("Ping command sent to all peers.")
	})
	myWindow.SetContent(container.NewVBox(thankLabel, pingButton))
}

func tidyUp() {
	fmt.Println("Exited")
}
