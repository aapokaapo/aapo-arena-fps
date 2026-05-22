package network

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings" // Added strings import
	"sync"
	"time"

	"github.com/quic-go/quic-go" // Added quic-go import
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"

	// Replace this with your actual module path
	"github.com/aapokaapo/aapo-arena-fps/GameServer/internal/game"
)

// TransportServer manages WebTransport connections and datagram routing
type TransportServer struct {
	world    *game.World
	server   *webtransport.Server

	mu       sync.RWMutex
	sessions map[string]*webtransport.Session
}

// NewTransportServer creates a new network layer linked to the game simulation
func NewTransportServer(world *game.World) *TransportServer {
	return &TransportServer{
		world:    world,
		sessions: make(map[string]*webtransport.Session),
	}
}

// Listen starts the HTTP/3 WebTransport server and the 60Hz broadcast loop
func (ts *TransportServer) Listen(addr, certFile, keyFile string) error {
	// FIX: Added the missing comma here
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("Failed to load certs: %v", err)
	}

	// Compute the certificate hash for the client
	certHash := sha256.Sum256(cert.Certificate[0])
	log.Printf("📜 Certificate SHA-256 hash (for client): %s", base64.RawStdEncoding.EncodeToString(certHash[:]))
	formattedArray := strings.ReplaceAll(fmt.Sprintf("%v", certHash[:]), " ", ", ")
	log.Printf("📜 As byte array: %s", formattedArray)

	// 1. Create a dedicated router
	mux := http.NewServeMux()
	
	// 2. Configure QUIC first
	quicConf := &quic.Config{
		EnableDatagrams:                  true,
		// EnableStreamResetPartialDelivery might be deprecated depending on your quic-go version, 
		// but if it compiles for you, leave it!
	}
	
	// 3. Create HTTP/3 server with TLS config
	h3Server := &http3.Server{
		Addr:       addr, // Make sure this uses the addr parameter passed in
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
	// Some versions of webtransport-go do this internally, but it's safe to keep if your version requires it.
	
	ts.server = wtServer

	// Route the WebTransport upgrade endpoint
	mux.HandleFunc("/fps", ts.handleWebTransport)

	// Start the 60Hz tick and snapshot broadcasting in a background goroutine
	go ts.broadcastLoop()

	log.Printf("WebTransport server listening on %s/fps", addr)
	
	// FIX: Since we already injected the certificates into TLSConfig up on line 60,
	// we just need to call ListenAndServe on the underlying H3 server.
	return ts.server.H3.ListenAndServe()
}

// handleWebTransport upgrades an HTTP/3 request to a WebTransport session
func (ts *TransportServer) handleWebTransport(w http.ResponseWriter, r *http.Request) {
	session, err := ts.server.Upgrade(w, r)
	if err != nil {
		log.Printf("WebTransport upgrade failed: %v", err)
		return
	}

	sessionID := r.RemoteAddr
	log.Printf("Client connected: %s", sessionID)

	ts.mu.Lock()
	ts.sessions[sessionID] = session
	ts.mu.Unlock()

	ts.world.AddPlayer(sessionID)

	go ts.sendWelcomeMessage(session, sessionID)

	ts.readDatagrams(session, sessionID)

	ts.mu.Lock()
	delete(ts.sessions, sessionID)
	ts.mu.Unlock()

	ts.world.RemovePlayer(sessionID)
	log.Printf("Client disconnected: %s", sessionID)
}

func (ts *TransportServer) sendWelcomeMessage(session *webtransport.Session, sessionID string) {
	stream, err := session.OpenStreamSync(context.Background())
	if err != nil {
		log.Printf("Failed to open welcome stream for %s: %v", sessionID, err)
		return
	}
	defer stream.Close()

	player := ts.world.GetPlayer(sessionID)
	if player == nil {
		return
	}

	payload, err := SerializeWelcomeMessage(sessionID, *player)
	if err != nil {
		log.Printf("Failed to serialize welcome message: %v", err)
		return
	}

	_, _ = stream.Write(payload)
}

func (ts *TransportServer) readDatagrams(session *webtransport.Session, sessionID string) {
	for {
		datagram, err := session.ReceiveDatagram(context.Background())
		if err != nil {
			break 
		}

		packet, err := DeserializeClientInputPacket(datagram)
		if err != nil {
			continue 
		}

		ts.world.ProcessClientInputs(sessionID, packet)
	}
}

func (ts *TransportServer) broadcastLoop() {
	tickRate := time.Second / 60
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	for range ticker.C {
		ts.world.Tick()

		// FIX: Correctly extract the three variables returned by GetSnapshotData()
		tick, timestamp, players := ts.world.GetSnapshotData()

		// FIX: Pass the three variables into SerializeSnapshot
		payload, err := SerializeSnapshot(tick, timestamp, players)
		if err != nil {
			log.Printf("Failed to serialize snapshot: %v", err)
			continue
		}

		ts.mu.RLock()
		for _, session := range ts.sessions {
			_ = session.SendDatagram(payload)
		}
		ts.mu.RUnlock()
	}
}
