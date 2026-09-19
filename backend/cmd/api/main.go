package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"network-scanner/backend/internal/controller"
	"network-scanner/backend/internal/database"
	"network-scanner/backend/internal/repository"
	"network-scanner/backend/internal/service"
)

func main() {
	// Beim Start wird zuerst die Verbindung zu PostgreSQL hergestellt.
	// Ohne Datenbankverbindung wird das Backend nicht gestartet.
	dbPool, err := database.NewPostgresPool(context.Background())
	if err != nil {
		log.Printf("Datenbankfehler: %v", err)
		return
	}
	defer dbPool.Close()

	log.Println("Verbindung zu PostgreSQL erfolgreich hergestellt")

	// Die Scanner-Adresse wird über eine Umgebungsvariable konfiguriert.
	// Innerhalb von Docker Compose wird http://scanner:8081 verwendet.
	// Für die lokale Entwicklung dient localhost als Standard.
	scannerURL := os.Getenv("SCANNER_URL")
	if scannerURL == "" {
		scannerURL = "http://localhost:8081"
	}

	// Aufbau der Anwendungsschichten.
	// Der ScannerService übernimmt die Kommunikation mit dem Scanner.
	scannerService := service.NewScannerService(scannerURL)

	// Das Repository ist für die Datenbankzugriffe verantwortlich.
	scanRepository := repository.NewScanRepository(dbPool)

	// Der ScanService verbindet Scanner-Kommunikation
	// und dauerhafte Speicherung in PostgreSQL.
	scanService := service.NewScanService(
		scannerService,
		scanRepository,
	)

	// Die Controller verarbeiten die HTTP-Anfragen des Clients.
	scannerController := controller.NewScannerController(scannerService)
	scanController := controller.NewScanController(scanService)

	mux := http.NewServeMux()

	// Prüft, ob das Backend grundsätzlich erreichbar ist.
	mux.HandleFunc(
		"GET /health",
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "backend ok")
		},
	)

	// Prüft die Verbindung zur getrennten Scanner-Komponente.
	mux.HandleFunc(
		"GET /api/scanner/health",
		scannerController.Health,
	)

	// Erstellt einen neuen asynchronen Scanauftrag.
	mux.HandleFunc(
		"POST /api/scans",
		scanController.CreateScan,
	)

	// Liefert die letzten 50 gespeicherten Scanläufe.
	mux.HandleFunc(
		"GET /api/scans",
		scanController.GetHistory,
	)

	// Liefert den aktuellen Status eines Scanauftrags.
	mux.HandleFunc(
		"GET /api/scans/{id}",
		scanController.GetScan,
	)

	// Liefert die strukturierten Hosts, Ports und Dienste
	// eines Scanlaufs aus der PostgreSQL-Datenbank.
	mux.HandleFunc(
		"GET /api/scans/{id}/results",
		scanController.GetResults,
	)

	log.Println("Backend läuft auf Port 8080")

	// Der HTTP-Server stellt die REST-Schnittstelle
	// für die spätere Weboberfläche bereit.
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Printf("HTTP-Server wurde beendet: %v", err)
	}
}
