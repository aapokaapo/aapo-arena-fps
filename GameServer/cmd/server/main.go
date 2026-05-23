package main

import (
	"flag"
	"log"
	// Replace with your actual GitHub module path
	"github.com/aapokaapo/aapo-arena-fps/GameServer/internal/game"
	"github.com/aapokaapo/aapo-arena-fps/GameServer/internal/network"
)

const PORT = ":4433"

func main() {
	recordMatch := flag.Bool("record", false, "Enable saving match replays to disk")
	matchID := flag.String("match-id", "dev_test", "The unique ID for this match")
	log.Println("Initializing Game Simulation...")

	// Parse the flags from the terminal
	flag.Parse()

	log.Println("Starting WebTransport Server on %v...", PORT)
	// 2. Pass the parsed flags into the World
	world := game.NewWorld(*recordMatch, *matchID)

	server := network.NewTransportServer(world)

	// Start the server (this blocks the main thread)
	if err := server.Listen(PORT, "server.crt", "server.key"); err != nil {
		log.Fatalf("Server crashed: %v", err)
	}
}
