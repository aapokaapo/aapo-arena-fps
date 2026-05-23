package network

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
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
	"github.com/aapokaapo/aapo-arena-fps/GameServer/internal/models"
)

// TransportServer manages WebTransport connections and datagram routing
type TransportServer struct {
	world  *game.World
	server *webtransport.Server

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
		EnableDatagrams: true,
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

	// 6. Configure HTTP/3 server for WebTransport (adds WebTransport settings)
	webtransport.ConfigureHTTP3Server(h3Server)

	ts.server = wtServer

	// Route the WebTransport upgrade endpoint
	mux.HandleFunc("/fps", ts.handleWebTransport)

	// Start the 60Hz tick and snapshot broadcasting in a background goroutine
	go ts.broadcastLoop()

	log.Printf("WebTransport server listening on %s/fps", addr)

	return ts.server.ListenAndServe()
}

// handleWebTransport upgrades an HTTP/3 request to a WebTransport session
func (ts *TransportServer) handleWebTransport(w http.ResponseWriter, r *http.Request) {
	log.Printf("WebTransport request: proto=%s method=%s url=%s", r.Proto, r.Method, r.URL)
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

	done := make(chan struct{}, 2)

	go ts.readDatagrams(session, sessionID, done)
	go ts.readStreams(session, sessionID, done)

	// Block here until one of those functions detects a disconnect
	<-done

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

// 1. Add the 'done' channel to the parameters
func (ts *TransportServer) readDatagrams(session *webtransport.Session, sessionID string, done chan struct{}) {
	for {
		datagram, err := session.ReceiveDatagram(context.Background())
		if err != nil {
			// 2. Send a signal to the channel instead of closing it
			done <- struct{}{}
			break
		}

		packet, err := DeserializeClientInputPacket(datagram)
		if err != nil {
			continue
		}

		ts.world.ProcessClientInputs(sessionID, packet)
	}
}

// readStreams listens for reliable events (like chat) coming from this client
func (ts *TransportServer) readStreams(session *webtransport.Session, sessionID string, done chan struct{}) {
	for {
		// AcceptUniStream blocks until the client opens a new stream to the server
		stream, err := session.AcceptUniStream(context.Background())
		if err != nil {
			// FIX: Send a signal instead of closing to prevent double-close panics
			done <- struct{}{}
			break
		}

		// Handle the stream in a new goroutine so we don't block other incoming streams
		go func(s webtransport.ReceiveStream) {
			// Read all the bytes the client sent in this stream
			data, err := io.ReadAll(s)
			if err != nil {
				return
			}

			// Pass the data to a router
			ts.handleIncomingReliableEvent(sessionID, data)
		}(stream)
	}
}

func (ts *TransportServer) broadcastLoop() {
	tickRate := time.Second / 60
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	for range ticker.C {
		ts.world.Tick()

		tick, timestamp, players := ts.world.GetSnapshotData()

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

// Incoming reliable events should be only ChatEvents but we can add others later if needed
func (ts *TransportServer) handleIncomingReliableEvent(senderID string, data []byte) {
	event, err := DeserializeServerEvent(data)
	if err != nil {
		log.Printf("Dropped malformed event stream from %s", senderID)
		return
	}

	switch event.Type {
	case models.EventTypeChat:
		chat, err := ParseChatPayload(event.Payload)
		if err != nil {
			return
		}

		chat.SenderID = senderID

		outgoingBytes, err := SerializeServerEvent(models.EventTypeChat, chat)
		if err != nil {
			return
		}

		// NEW: Reconstruct the event envelope so the World can save it
		replayEvent := models.ServerEvent{
			Type:    models.EventTypeChat,
			Payload: outgoingBytes,
		}
		ts.world.AddEventToReplay(replayEvent) // Send to the recorder!

		// Broadcast to live players
		ts.BroadcastReliable(outgoingBytes)
	}
}

func (ts *TransportServer) BroadcastReliable(data []byte) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	for id, session := range ts.sessions {
		// Optimization: Spin up a micro-thread for each client so one laggy
		// player doesn't freeze the entire broadcast loop!
		go func(sess *webtransport.Session, peerID string) {
			// We can also add a 2-second timeout to the context so it doesn't hang forever
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			stream, err := sess.OpenStreamSync(ctx)
			if err != nil {
				log.Printf("Failed to open stream to %s: %v", peerID, err)
				return
			}
			defer stream.Close()

			_, _ = stream.Write(data)
		}(session, id)
	}
}
