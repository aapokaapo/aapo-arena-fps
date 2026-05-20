package main

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

// Global Server State Manager
type GameServer struct {
	clients   map[string]*webtransport.Session
	clientsMu sync.RWMutex
	tickCount uint64
}

func main() {
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatalf("Failed to load certs: %v", err)
	}

	// Compute the certificate hash for the client
	certHash := sha256.Sum256(cert.Certificate[0])
	log.Printf("📜 Certificate SHA-256 hash (for client): %s", base64.RawStdEncoding.EncodeToString(certHash[:]))
	formattedArray := strings.ReplaceAll(fmt.Sprintf("%v", certHash[:]), " ", ", ")
	log.Printf("📜 As byte array: %s", formattedArray)

	// Initialize global server state
	server := &GameServer{
		clients: make(map[string]*webtransport.Session),
	}

	// 1. Create a dedicated router
	mux := http.NewServeMux()

	// 2. Configure QUIC first
	quicConf := &quic.Config{
		EnableDatagrams:                  true,
		EnableStreamResetPartialDelivery: true,
	}

	// 3. Create HTTP/3 server with TLS config
	h3Server := &http3.Server{
		Addr:       ":4433",
		Handler:    mux,
		QUICConfig: quicConf,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}

	// 4. Configure TLS for HTTP/3 (this adds NextProtos)
	h3Server.TLSConfig = http3.ConfigureTLSConfig(h3Server.TLSConfig)

	// 5. Create WebTransport server
	wtServer := &webtransport.Server{
		H3: h3Server,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// 6. Configure HTTP3 server for WebTransport (adds WebTransport settings)
	webtransport.ConfigureHTTP3Server(wtServer.H3)

	// 7. Register the WebTransport upgrade endpoint
	mux.HandleFunc("/game-server", func(w http.ResponseWriter, r *http.Request) {
		session, err := wtServer.Upgrade(w, r)
		if err != nil {
			log.Printf("❌ Upgrade Failed: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Handle registration and client updates
		server.registerClient(session)
	})

	// START THE SERVER TICK LOOP IN THE BACKGROUND
	go server.startTickLoop(16666 * time.Microsecond) // ~60Hz (16.66ms per frame)

	log.Printf("🚀 WebTransport Game Server running on UDP port 4433 at 60Hz")
	log.Printf("⚠️  Copy the certificate hash above to your Godot client script!")

	// 8. Start the server
	err = wtServer.ListenAndServe()
	if err != nil {
		log.Fatalf("Critical server error: %v", err)
	}
}

// registerClient safely maps a new session and launches an input listener
func (s *GameServer) registerClient(session *webtransport.Session) {
	// Generate a unique identifier for this peer connection
	clientID := fmt.Sprintf("player_%d", time.Now().UnixNano())

	s.clientsMu.Lock()
	s.clients[clientID] = session
	s.clientsMu.Unlock()

	log.Printf("🎮 Client Connected: %s (Total Active: %d)", clientID, s.countClients())

	// Handle the reader thread for this specific client
	go func() {
		defer func() {
			s.clientsMu.Lock()
			delete(s.clients, clientID)
			s.clientsMu.Unlock()
			session.CloseWithError(0, "disconnected")
			log.Printf("❌ Client Disconnected: %s (Total Active: %d)", clientID, s.countClients())
		}()

		ctx := session.Context()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := session.ReceiveDatagram(ctx)
				if err != nil {
					return
				}

				log.Printf("📥 Received from %s: %s", clientID, string(msg))
			}
		}
	}()
}

// startTickLoop runs a fixed ticker interval to process simulations and send snapshots
func (s *GameServer) startTickLoop(rate time.Duration) {
	ticker := time.NewTicker(rate)
	defer ticker.Stop()

	log.Printf("⏳ Server simulation engine initialized.")

	for range ticker.C {
		s.tickCount++
		s.broadcastSnapshot()
	}
}

// broadcastSnapshot serializes the current world frame state and sends it to everyone via UDP
func (s *GameServer) broadcastSnapshot() {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	if len(s.clients) == 0 {
		return
	}

	snapshotPayload := []byte(fmt.Sprintf("SNAPSHOT|tick:%d|time:%d", s.tickCount, time.Now().UnixMilli()))

	for id, session := range s.clients {
		err := session.SendDatagram(snapshotPayload)
		if err != nil {
			log.Printf("⚠️  Snapshot drop on peer %s: %v", id, err)
		}
	}
}

func (s *GameServer) countClients() int {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	return len(s.clients)
}
