package main

import (
	"fmt"
	"log"
	"net/http"

	"network-scanner/scanner/internal/controller"
	"network-scanner/scanner/internal/nmap"
	"network-scanner/scanner/internal/service"
)

func main() {
	// Die einzelnen Komponenten werden beim Start erstellt
	// und über ihre Abhängigkeiten miteinander verbunden.
	nmapExecutor := nmap.NewExecutor()
	scanService := service.NewScanService(nmapExecutor)
	scanController := controller.NewScanController(scanService)

	mux := http.NewServeMux()

	// Health-Endpunkt zur einfachen Prüfung der Scanner-Komponente.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "scanner ok")
	})

	// Endpunkte zum Erstellen und Abfragen von Scanaufträgen.
	mux.HandleFunc("POST /scan", scanController.CreateScan)
	mux.HandleFunc("GET /scan/{id}", scanController.GetScan)

	log.Println("Scanner läuft auf Port 8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}
