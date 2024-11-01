package gui

import (
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
			log.Println("Address inputed: ", inputedAddress.Text)
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
