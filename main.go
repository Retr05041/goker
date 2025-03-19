package main

import "goker/internal/gamemanager"

func main() {
	manager := new(gamemanager.GameManager)
	manager.StartGame()
}
