package gamemanager

import (
	"fmt"
	"goker/internal/channelmanager"
	"log"
	"strings"

	"fyne.io/fyne/v2/canvas"
	//"github.com/chehsunliu/poker"
)

// Setup board with back of cards
func (gm *GameManager) initBoard() {
	for i := 0; i < 5; i++ {
		cardImage := canvas.NewImageFromFile("media/svg_playing_cards/backs/png_96_dpi/red.png")
		cardImage.FillMode = canvas.ImageFillOriginal

		gm.Board = append(gm.Board, cardImage)
	}

	channelmanager.TGUI_BoardChan <- gm.Board
}

func (gm *GameManager) EvaluateHands() {

	for id := range gm.state.Players {
		if id == gm.state.Me {
			continue
		}
		hand, exists := gm.network.OthersHands[id]
		if !exists || len(hand) == 0 {
			log.Printf("Error: No cards found for peer %s in OthersHands", id)
			return
		}
		cardOneName, exists := gm.network.Deck.GetCardFromRefDeck(hand[0].CardValue)
		cardTwoName, exists1 := gm.network.Deck.GetCardFromRefDeck(hand[1].CardValue)
		if exists && exists1 {
			fmt.Println(gm.state.Players[id] + " cards: " + cardOneName + ", " + cardTwoName)
		}

	}

	// TODO: Implement
	//deck := poker.NewDeck()
	//var hands map[peer.ID][]poker.Card
	//hands = make(map[peer.ID][]poker.Card)
	//fmt.Println(hand)

	//rank := poker.Evaluate(hand)
	//fmt.Println(rank)
	//fmt.Println(poker.RankString(rank))
}

var suitMap = map[string]byte{
	"clubs":    'c',
	"diamonds": 'd',
	"hearts":   'h',
	"spades":   's',
}

var rankMap = map[string]string{
	"2": "2", "3": "3", "4": "4", "5": "5", "6": "6", "7": "7", "8": "8", "9": "9", "10": "10",
	"jack": "J", "queen": "Q", "king": "K", "ace": "A",
}

func convertMyCardStringsToLibrarys(myCardStrings []string) string {
	var converted []string
	for _, card := range myCardStrings {
		parts := strings.Split(card, "_")
		if len(parts) != 2 {
			continue
		}
		suit, suitExists := suitMap[parts[0]]
		rank, rankExists := rankMap[parts[1]]
		if suitExists && rankExists {
			converted = append(converted, fmt.Sprintf("%s%c", rank, suit))
		}
	}
	return strings.Join(converted, " ")
}
