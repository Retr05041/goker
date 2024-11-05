package gui

import (
	"goker/internal/p2p"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

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
			go p2p.Init(false, inputedAddress.Text)
		} else if choice == "Host" {
			go p2p.Init(true, "")
		}

		thanksLabel := widget.NewLabel("Thanks!")
		myWindow.SetContent(container.NewCenter(thanksLabel))
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
