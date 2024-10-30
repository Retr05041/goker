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

	check := widget.NewCheck("Optional", func(value bool) {
		log.Println("Check set to", value)
	})
	radio := widget.NewRadioGroup([]string{"Option 1", "Option 2"}, func(value string) {
		log.Println("Radio set to", value)
	})
	combo := widget.NewSelect([]string{"Option 1", "Option 2"}, func(value string) {
		log.Println("Select set to", value)
	})

	myWindow.SetContent(container.NewVBox(check, radio, combo))
	myWindow.Show()
	myApp.Run()
	tidyUp()
}


func tidyUp() {
	log.Println("Exited")
}
