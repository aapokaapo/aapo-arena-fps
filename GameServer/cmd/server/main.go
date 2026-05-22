package main

import (
	"log"
	// Replace with your actual GitHub module path
	"github.com/aapokaapo/aapo-arena-fps/GameServer/internal/game"
	"github.com/aapokaapo/aapo-arena-fps/GameServer/internal/network"
)

func main() {
	log.Println("Initializing Game Simulation...")
	world := game.NewWorld()

	log.Println("Starting WebTransport Server on :4433...")
	server := network.NewTransportServer(world)
	
	// Start the server (this blocks the main thread)
	if err := server.Listen(); err != nil {
		log.Fatalf("Server crashed: %v", err)
	}
}
