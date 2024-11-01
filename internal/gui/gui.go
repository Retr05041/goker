package gui

import (
	"goker/internal/p2p"
	"log"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func Init() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Choice Widgets")


	inputedAddress := widget.NewEntry()
	inputedAddress.SetPlaceHolder("Host address...")
	inputedAddress.Disable()
	
	var choice string
	peerType := widget.NewRadioGroup([]string{"Host", "Connect"}, func(value string) {
		if value == "Connect" {
			inputedAddress.Enable()
		} else {
			inputedAddress.Disable()
		}
		choice = value
	})

	submit := widget.NewButton("Submit", func() {
		log.Println("Choice made: ", choice)
		if choice == "Connect" {
			p2p.Init(false, inputedAddress.Text)
		} else if choice == "Host" {
			p2p.Init(true, "")
		}
	})


	myWindow.SetContent(container.NewVBox(peerType, inputedAddress, submit))
	myWindow.Show()
	myApp.Run()
	tidyUp()
}


func tidyUp() {
	log.Println("Exited")
}
