package gamemanager

import (
	"fmt"
	"goker/internal/channelmanager"
	"goker/internal/p2p"
	"log"
	"strings"

	"fyne.io/fyne/v2/canvas"
	"github.com/chehsunliu/poker"
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
	flopCardOne, flop1Exists := gm.network.Deck.GetCardFromRefDeck(gm.network.Flop[0].CardValue)
	flopCardTwo, flop2Exists := gm.network.Deck.GetCardFromRefDeck(gm.network.Flop[1].CardValue)
	flopCardThree, flop3Exists := gm.network.Deck.GetCardFromRefDeck(gm.network.Flop[2].CardValue)
	if !(flop1Exists && flop2Exists && flop3Exists) {
		fmt.Println("flop cards didn't exist.")
	}

	turnCard, turnExists := gm.network.Deck.GetCardFromRefDeck(gm.network.Turn.CardValue)
	if !turnExists {
		fmt.Println("turn card didn't exist.")
	}

	riverCard, riverExists := gm.network.Deck.GetCardFromRefDeck(gm.network.River.CardValue)
	if !riverExists {
		fmt.Println("river card didn't exist.")
	}

	var bestPlayer string
	var bestRank int32
	bestRank = 10000 // Since the lower the rank the better the hand

	IDs := gm.state.GetTurnOrder()
	for _, id := range IDs {
		var hand []p2p.CardInfo
		if id == gm.network.ThisHost.ID() {
			hand = gm.network.MyHand
			if len(hand) != 2 {
				log.Println("Error: No cards found for me!")
				return
			}
		} else {
			OthersHand, exists := gm.network.OthersHands[id]
			if !exists || len(OthersHand) == 0 {
				log.Printf("Error: No cards found for peer %s in OthersHands", id)
				return
			}
			hand = OthersHand
		}

		// Calc best hand
		cardOneName, exists := gm.network.Deck.GetCardFromRefDeck(hand[0].CardValue)
		cardTwoName, exists1 := gm.network.Deck.GetCardFromRefDeck(hand[1].CardValue)
		fullHand := []string{flopCardOne, flopCardTwo, flopCardThree, turnCard, riverCard, cardOneName, cardTwoName}
		fmt.Println(fullHand)

		if exists && exists1 {
			currHand := convertMyCardStringsToLibrarys(fullHand)
			rank := poker.Evaluate(currHand)
			if rank < bestRank {
				bestPlayer = gm.state.Players[id]
				bestRank = rank
			}
			fmt.Println(gm.state.Players[id] + " got " + poker.RankString(rank))
		} else {
			fmt.Println(gm.state.Players[id] + " cards didn't exist.")
		}
	}

	fmt.Print(bestPlayer + " won with: ")
	fmt.Println(poker.RankString(bestRank))
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

func convertMyCardStringsToLibrarys(myCardStrings []string) []poker.Card {
	var converted []poker.Card
	for _, card := range myCardStrings {
		parts := strings.Split(card, "_")
		if len(parts) != 2 {
			continue
		}
		suit, suitExists := suitMap[parts[0]]
		rank, rankExists := rankMap[parts[1]]
		if suitExists && rankExists {
			converted = append(converted, poker.NewCard(fmt.Sprintf("%s%c", rank, suit)))
		}
	}

	return converted
}
