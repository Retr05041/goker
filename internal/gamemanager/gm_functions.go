package gamemanager

import (
	"fmt"
	"goker/internal/channelmanager"
	"goker/internal/p2p"
	"log"
	"strings"

	"fyne.io/fyne/v2/canvas"
	"github.com/chehsunliu/poker"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Setup board with back of cards
func (gm *GameManager) initBoard() {
	gm.Board = nil
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

	var bestID peer.ID
	var bestRank int32
	bestRank = 10000 // Since the lower the rank the better the hand

	IDs := gm.state.GetTurnOrder()
	for _, id := range IDs {
		if !gm.state.FoldedPlayers[id] { // Don't want to add folded players to our check
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
					bestID = id
					bestRank = rank
				}
				fmt.Println(gm.state.Players[id] + " got " + poker.RankString(rank))
			} else {
				fmt.Println(gm.state.Players[id] + " cards didn't exist.")
			}
		}
	}

	gm.state.Winner = bestID
	gm.RestartRound()
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

// RestartRound will be called when a game is being played and the round is over.
// Will distribute pot and reset phase bets and restart the protocol
func (gm *GameManager) RestartRound() {
	// Distribute the pot to the winner
	pot := gm.state.GetCurrentPot()
	if winner, exists := gm.state.PlayersMoney[gm.state.Winner]; exists {
		gm.state.PlayersMoney[gm.state.Winner] = winner + pot
		log.Printf("%s won the pot of %.2f!", gm.state.Players[gm.state.Winner], pot)
	} else {
		log.Println("Error: Winner not found in player money map")
	}

	// Reset round state
	for id := range gm.state.BetHistory {
		gm.state.BetHistory[id] = 0.0
	}
	for id := range gm.state.FoldedPlayers {
		gm.state.FoldedPlayers[id] = false
	}
	for id := range gm.state.PlayedThisPhase {
		gm.state.PlayedThisPhase[id] = false
	}

	gm.state.MyBet = 0.0
	gm.state.Phase = "preflop"
	gm.state.WhosTurn = 0
	gm.network.OthersHands = make(map[peer.ID][]p2p.CardInfo) // Need to reset this

	// Reinitialize the board
	gm.initBoard()

	// Notify GUI to update
	channelmanager.TGUI_PotChan <- 0.0
	channelmanager.TGUI_PlayerInfo <- gm.state.GetPlayerInfo()

	gm.network.Deck.GenerateDecks("gokerdecksecretkeyforhashesversion1")

	// Start the next round's protocol if you are the host - Host will be first in turn order always
	if gm.network.ThisHost.ID() == gm.state.TurnOrder[0] {
		gm.RunProtocol()
	}
}

// Run through setting up keyring, shuffling deck, and dealing
func (gm *GameManager) RunProtocol() {
	// Setup keyring for this round
	gm.network.Keyring.GeneratePQ()
	gm.network.Keyring.GenerateKeys()

	// Setup deck
	gm.network.ExecuteCommand(&p2p.SendPQCommand{})            // Send everyone the generated P and Q so they can setup their Keyring
	gm.network.ExecuteCommand(&p2p.ProtocolFirstStepCommand{}) // Setting up deck pt.1
	gm.network.ExecuteCommand(&p2p.BroadcastNewDeck{})
	gm.network.ExecuteCommand(&p2p.ProtocolSecondStepCommand{}) // Setting up deck pt.2 & Sets everyones hands
	gm.network.ExecuteCommand(&p2p.BroadcastDeck{})

	// Setup hands
	gm.network.ExecuteCommand(&p2p.CanRequestHand{}) // Deals hands one player at at time
	gm.network.ExecuteCommand(&p2p.RequestHandCommand{})
}
