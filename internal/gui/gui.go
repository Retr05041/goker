package gui

import (
	"goker/internal/p2p"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var server p2p.BootstrapServer

func Init() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Choice Widgets")

	var choice string

	inputedAddress := widget.NewEntry()
	inputedAddress.SetPlaceHolder("Host address...")
	inputedAddress.Disable()

	submit := widget.NewButton("Submit", func() {
		log.Println("Choice made: ", choice)
		if choice == "Connect" {
			server.Init(false, inputedAddress.Text)
			thankLabel := widget.NewLabel("Thanks!")
			myWindow.SetContent(container.NewCenter(thankLabel))
		} else if choice == "Host" {
			server.Init(true, "")
			copyAddrButton := widget.NewButton("Copy server address", func() {
				myWindow.Clipboard().SetContent(server.HostMultiaddr)
			})
			myWindow.SetContent(copyAddrButton)
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


func tidyUp() {
	log.Println("Exited")
}
