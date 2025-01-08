package gui

import (
	"fmt"

	"goker/internal/channelmanager"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Main Menu
func showMenuUI(givenWindow fyne.Window) {
	var hostOrConnect string
	banner := canvas.NewText("Goker", BLUE)
	banner.TextSize = 24
	banner.TextStyle = fyne.TextStyle{Bold: true, Italic: false}
	banner.Alignment = fyne.TextAlignCenter

	inputedAddress := widget.NewEntry()
	inputedAddress.SetPlaceHolder("Host address...")
	inputedAddress.Disable()

	submit := widget.NewButton("Submit", func() {
		fmt.Println("Choice made: ", hostOrConnect)
		if hostOrConnect == "Connect" {
			channelmanager.ActionChannel <- channelmanager.ActionType{Action: "hostOrConnectPressed", DataS: &inputedAddress.Text}
			showConnectedUI(givenWindow)
		} else if hostOrConnect == "Host" {
			channelmanager.ActionChannel <- channelmanager.ActionType{Action: "hostOrConnectPressed", DataS: nil}
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

// Host UI is the same as connectedUI but without settings
func showHostUI(givenWindow fyne.Window) {
	playButton := widget.NewButton("Play", func() {
		showGameScreen(givenWindow)
	})
	copyAddrButton := widget.NewButton("Copy server address", func() {
		givenWindow.Clipboard().SetContent(hostAddress)
	})

	givenWindow.SetContent(
		container.NewCenter(
			container.NewVBox(
				numOfPlayers,
				copyAddrButton,
				playButton)))
}

// Connected UI is just a waiting area for the host to start
func showConnectedUI(myWindow fyne.Window) {
	waiting := widget.NewLabel("Waiting for host to begin game!")
	myWindow.SetContent(
		container.NewCenter(waiting))
}

// Main game screen
func showGameScreen(givenWindow fyne.Window) {

	foldButton := widget.NewButton("Fold", func() {
		channelmanager.ActionChannel <- channelmanager.ActionType{Action: "Fold"}
	})
	raiseButton := widget.NewButton("Raise", func() {
		channelmanager.ActionChannel <- channelmanager.ActionType{Action: "Raise", DataF: &betSlider.Value}
	})
	callButton := widget.NewButton("Call", func() {
		channelmanager.ActionChannel <- channelmanager.ActionType{Action: "Call"}
	})
	checkButton := widget.NewButton("Check", func() {
		channelmanager.ActionChannel <- channelmanager.ActionType{Action: "Check"}
	})

	givenWindow.SetContent(
		container.NewCenter(
			container.NewVBox(
				container.NewCenter(potLabel),
				boardGrid,
				container.NewPadded(
					container.NewCenter(
						container.NewVBox(
							container.NewCenter(moneyLabel),
							container.NewHBox(
								handGrid,
								container.NewVBox(
									foldButton,
									callButton,
									container.NewHBox(raiseButton, valueLabel),
									betSlider),
								checkButton)))))))
}
