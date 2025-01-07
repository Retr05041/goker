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

	inputedAddress := widget.NewEntry()
	inputedAddress.SetPlaceHolder("Host address...")
	inputedAddress.Disable()


	banner := canvas.NewText("Goker", BLUE)
	banner.TextSize = 24
	banner.TextStyle = fyne.TextStyle{Bold: true, Italic: false}
	banner.Alignment = fyne.TextAlignCenter

	submit := widget.NewButton("Submit", func() {
		fmt.Println("Choice made: ", hostOrConnect)
		channelmanager.HostConnectChannel <- hostOrConnect
		if hostOrConnect == "Connect" {
			showConnectedUI(givenWindow)
		} else if hostOrConnect == "Host" {
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
	thankLabel := widget.NewLabel("Playing as Host!")

	givenWindow.SetContent(
		container.NewVBox(thankLabel))
}

// Connected UI is just a waiting area for the host to start
func showConnectedUI(myWindow fyne.Window) {
	thankLabel := widget.NewLabel("Connected to Host!")

	myWindow.SetContent(
		container.NewVBox(thankLabel))
}

// Main game screen
func showGameScreen(givenWindow fyne.Window) {

	foldButton := widget.NewButton("Fold", func() {
		channelmanager.ActionChannel <- channelmanager.ActionType{
			Action: "Fold",
			Data: nil,
		}
	})
	raiseButton := widget.NewButton("Raise", func() {
		channelmanager.ActionChannel <- channelmanager.ActionType{
			Action: "Raise",
			Data: &betSlider.Value,
		}
	})
	callButton := widget.NewButton("Call", func() {
		channelmanager.ActionChannel <- channelmanager.ActionType{
			Action: "Call",
			Data: nil,
		}
	})
	checkButton := widget.NewButton("Check", func() {
		channelmanager.ActionChannel <- channelmanager.ActionType{
			Action: "Check",
			Data: nil,
		}
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
