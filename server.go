package server

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/context"

	"golang.org/x/crypto/acme/autocert"
)

// StartDev will start a development server.
func StartDev(h http.Handler, addr, certFile, keyFile string) {
	srv := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		Addr:         addr,
		Handler:      h,
	}

	log.Printf("Running development server on: https://%s\n", addr)

	go graceful(srv, 30*time.Second)

	log.Fatal(srv.ListenAndServeTLS(certFile, keyFile))
}

// StartProd will start a production server with automated certificates via LetsEncrypt.
func StartProd(h http.Handler, hosts ...string) {
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(hosts...),
		Cache:      autocert.DirCache("certs"),
	}

	srv := &http.Server{
		Addr:         ":https",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
		Handler: h,
	}

	go func() {
		log.Fatal(http.ListenAndServe(":http", certManager.HTTPHandler(nil)))
	}()

	log.Println("Running production server...")

	go graceful(srv, 60*time.Second)

	log.Fatal(srv.ListenAndServeTLS("", ""))
}

func graceful(srv *http.Server, timeout time.Duration) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Printf("Server shutting down, timeout: %s", timeout)
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error: %v\n", err)
	} else {
		log.Println("Server stopped")
	}
}
