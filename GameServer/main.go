package main

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"net/http"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

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

	// 1. Create a dedicated router
	mux := http.NewServeMux()

	// 2. Configure TLS with proper ALPN protocol
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{http3.NextProtoH3},
	}

	// 3. Configure the server with QUIC datagram support
	wtServer := &webtransport.Server{
		H3: &http3.Server{
			Addr:      ":4433",
			Handler:   mux,
			TLSConfig: tlsConfig,
			QUICConfig: &quic.Config{
				EnableDatagrams: true,
			},
		},
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// 4. Configure HTTP3 for WebTransport
	webtransport.ConfigureHTTP3Server(wtServer.H3)

	// 5. Register the WebTransport upgrade endpoint
	mux.HandleFunc("/game-server", func(w http.ResponseWriter, r *http.Request) {
		session, err := wtServer.Upgrade(w, r)
		if err != nil {
			log.Printf("❌ Upgrade Failed: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Printf("🎮 Client Connected!")

		// Handle this session in a goroutine
		go handleClient(session)
	})

	log.Printf("🚀 WebTransport Game Server running on UDP port 4433")
	log.Printf("⚠️  Copy the certificate hash above to your Godot client script!")

	// 6. Start the server
	err = wtServer.ListenAndServe()
	if err != nil {
		log.Fatalf("Critical server error: %v", err)
	}
}

// handleClient manages a single client connection
func handleClient(session *webtransport.Session) {
	defer session.CloseWithError(0, "disconnected")

	ctx := session.Context()
	for {
		select {
		case <-ctx.Done():
			log.Printf("❌ Client disconnected (context closed)")
			return
		default:
			msg, err := session.ReceiveDatagram(ctx)
			if err != nil {
				log.Printf("❌ Client Disconnected: %v", err)
				return
			}

			log.Printf("📥 Received: %s", string(msg))
			response := []byte(fmt.Sprintf("Echo at %d: %s", time.Now().UnixMilli(), string(msg)))
			
			if err := session.SendDatagram(response); err != nil {
				log.Printf("❌ Failed to send response: %v", err)
				return
			}
		}
	}
}
