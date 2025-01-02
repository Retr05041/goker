package gui

import (

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

func showMenuUI(myWindow fyne.Window) {
	myWindow.SetContent(
		container.NewCenter(
			container.NewGridWrap(
				fyne.NewSize(float32(MAX_WIDTH), float32(MAX_HEIGHT)), 
				container.NewVBox(banner, peerType, inputedAddress, submit))))
}

// What the host will see
func showHostUI(myWindow fyne.Window) {
	myWindow.SetContent(
		container.NewVBox(copyAddrButton, testProtocolFS, testProtocolSS, DisplayDeck))
}

// What a connected peer will see
func showConnectedUI(myWindow fyne.Window) {
	myWindow.SetContent(
		container.NewVBox(thankLabel, DisplayDeck))
}

func showGameScreen(givenWindow fyne.Window) {
	// Call setupCards to initialize the playingCards map
	loadCard("spades_ace")
	loadCard("hearts_ace")

	// Create a container to hold all the card images in a grid
	grid := container.NewGridWithColumns(2) // Adjust the number of columns as desired

	// Add each card image to the grid
	for _, cardImage := range myHand {
		grid.Add(cardImage)
	}

	// Center the grid in the window
	givenWindow.SetContent(
		container.NewCenter(
			container.NewHBox(grid, container.NewVBox(foldButton, raiseButton, callButton, checkButton))))
}
