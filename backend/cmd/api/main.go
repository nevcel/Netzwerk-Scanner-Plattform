package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"network-scanner/backend/internal/controller"
	"network-scanner/backend/internal/service"
)

func main() {
	// Die Adresse der Scanner-Komponente wird über eine
	// Umgebungsvariable konfiguriert. Für die lokale Entwicklung
	// wird localhost als Standard verwendet.
	scannerURL := os.Getenv("SCANNER_URL")
	if scannerURL == "" {
		scannerURL = "http://localhost:8081"
	}

	// Die benötigten Services und Controller werden beim Start
	// erstellt und miteinander verbunden.
	scannerService := service.NewScannerService(scannerURL)

	scannerController := controller.NewScannerController(scannerService)
	scanController := controller.NewScanController(scannerService)

	mux := http.NewServeMux()

	// Einfacher Health-Endpunkt zur Prüfung,
	// ob das Backend grundsätzlich erreichbar ist.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "backend ok")
	})

	// Prüft die Verbindung zwischen Backend
	// und der getrennten Scanner-Komponente.
	mux.HandleFunc(
		"GET /api/scanner/health",
		scannerController.Health,
	)

	// API-Endpunkte zum Starten und Abfragen
	// von asynchronen Scanaufträgen.
	mux.HandleFunc(
		"POST /api/scans",
		scanController.CreateScan,
	)

	mux.HandleFunc(
		"GET /api/scans/{id}",
		scanController.GetScan,
	)

	log.Println("Backend läuft auf Port 8080")

	// Der HTTP-Server stellt die REST-Schnittstelle
	// des Backends auf Port 8080 bereit.
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
