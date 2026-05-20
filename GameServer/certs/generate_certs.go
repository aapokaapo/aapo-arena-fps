package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

func main() {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Godot WebTransport"}},
		// THE FIX: Backdate by 24 hours to prevent all clock-skew issues
		NotBefore:    time.Now().Add(-24 * time.Hour), 
		NotAfter:     time.Now().Add(10 * 24 * time.Hour), 
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}

	certBytes, _ := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	
	certOut, _ := os.Create("server.crt")
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	certOut.Close()

	keyBytes, _ := x509.MarshalECPrivateKey(priv)
	keyOut, _ := os.OpenFile("server.key", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	keyOut.Close()

	hash := sha256.Sum256(certBytes)
	
	fmt.Println("✅ Backdated ECDSA Certificates generated successfully!")
	fmt.Printf("\nconst certHash = new Uint8Array([%d", hash[0])
	for i := 1; i < len(hash); i++ {
		fmt.Printf(", %d", hash[i])
	}
	fmt.Println("]);\n")
}
