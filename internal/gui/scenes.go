package gui

import (
	"goker/internal/channelmanager"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func setWindowContent(window fyne.Window, content fyne.CanvasObject) {
	if window.Content() != content {
		window.SetContent(content)
		window.Resize(fyne.NewSize(float32(MAX_WIDTH), float32(MAX_HEIGHT))) // Ensure size consistency
	}
}

// Main Menu
func showMenuUI(givenWindow fyne.Window) {
	banner := canvas.NewText("Goker", BLUE)
	banner.TextSize = 32
	banner.TextStyle = fyne.TextStyle{Bold: true, Italic: false}
	banner.Alignment = fyne.TextAlignCenter

	nickname := widget.NewEntry()
	nickname.SetPlaceHolder("Nickname...")

	inputedAddress := widget.NewEntry()
	inputedAddress.SetPlaceHolder("Host address...")

	host := widget.NewButton("Host", func() {
		if nickname.Text != "" {
			channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "hostOrConnectPressed", DataS: []string{nickname.Text}}
			showHostUI(givenWindow)
		}
	})

	connect := widget.NewButton("Connect", func() {
		if nickname.Text != "" {
			if inputedAddress.Text != "" {
				channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "hostOrConnectPressed", DataS: []string{nickname.Text, inputedAddress.Text}}
				showConnectedUI(givenWindow)
			}
		}
	})

	setWindowContent(givenWindow,
		container.NewCenter(
			container.NewGridWrap(
				fyne.NewSize(float32(MAX_WIDTH)/2, float32(MAX_HEIGHT)/2),
				container.NewVBox(banner, nickname, host, container.NewGridWithColumns(2, connect, inputedAddress)))))
}

// Host UI is the same as connectedUI but without settings
func showHostUI(givenWindow fyne.Window) {
	playButton := widget.NewButton("Play", func() {
		channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "startRound"}
	})
	copyLBAddrButton := widget.NewButton("Copy LB address", func() {
		givenWindow.Clipboard().SetContent(loopbackAddress)
	})
	copyLNAddrButton := widget.NewButton("Copy LAN address", func() {
		givenWindow.Clipboard().SetContent(lanAddress)
	})

	setWindowContent(givenWindow,
		container.NewCenter(
			container.NewVBox(
				numOfPlayers,
				container.NewHBox(copyLBAddrButton, copyLNAddrButton),
				playButton)))
}

// Connected UI is just a waiting area for the host to start
func showConnectedUI(givenWindow fyne.Window) {
	waiting := widget.NewLabel("Waiting for host to begin game!")
	setWindowContent(givenWindow,
		container.NewCenter(
			container.NewVBox(numOfPlayers, waiting)))
}

// Main game screen
func showGameScreen(givenWindow fyne.Window) {
	foldButton := widget.NewButton("Fold", func() {
		channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "Fold"}
	})
	raiseButton := widget.NewButton("Raise", func() {
		if betSlider.Value <= myMoney {
			channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "Raise", DataF: betSlider.Value}
		}
	})
	callButton := widget.NewButton("Call", func() {
		channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "Call"}
	})
	checkButton := widget.NewButton("Check", func() {
		channelmanager.FGUI_ActionChan <- channelmanager.ActionType{Action: "Check"}
	})

	setWindowContent(givenWindow,
		container.NewBorder(
			nil,
			nil,
			playerCards,
			container.NewCenter(
				container.NewVBox(
					container.NewCenter(potLabel),
					boardGrid,
					container.NewCenter(
						container.NewHBox(
							handGrid,
							container.NewVBox(
								foldButton,
								callButton,
								container.NewHBox(raiseButton, valueLabel),
								betSlider),
							checkButton))))))
}
