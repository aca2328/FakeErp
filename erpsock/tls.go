package main

// HTTPS + HTTP/2 listener on :8443, to demo re-encryption to the pool.
// Uses TLS_CERT/TLS_KEY if set, otherwise a self-signed certificate.

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"
)

func selfSigned() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: hostname, Organization: []string{"PRIMO SAS " + siteCode}},
		DNSNames:     []string{hostname, "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

func serveTLS(h http.Handler) {
	var (
		cert tls.Certificate
		err  error
	)
	if c, k := os.Getenv("TLS_CERT"), os.Getenv("TLS_KEY"); c != "" && k != "" {
		cert, err = tls.LoadX509KeyPair(c, k)
	} else {
		cert, err = selfSigned()
	}
	if err != nil {
		log.Fatal("TLS certificate: ", err)
	}
	srv := &http.Server{
		Addr:              ":8443",
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		TLSConfig:         &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	log.Fatal(srv.ListenAndServeTLS("", ""))
}
