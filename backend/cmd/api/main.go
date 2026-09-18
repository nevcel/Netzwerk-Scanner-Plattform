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
	// Für die lokale Entwicklung wird localhost verwendet.
	scannerURL := os.Getenv("SCANNER_URL")
	if scannerURL == "" {
		scannerURL = "http://localhost:8081"
	}

	// Aufbau der einzelnen Anwendungsschichten.
	scannerService := service.NewScannerService(scannerURL)

	scanRepository := repository.NewScanRepository(dbPool)

	scanService := service.NewScanService(
		scannerService,
		scanRepository,
	)

	scannerController := controller.NewScannerController(scannerService)
	scanController := controller.NewScanController(scanService)

	mux := http.NewServeMux()

	// Health-Endpunkt zur Prüfung des Backends.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "backend ok")
	})

	// Prüft die Erreichbarkeit der getrennten Scanner-Komponente.
	mux.HandleFunc(
		"GET /api/scanner/health",
		scannerController.Health,
	)

	// API-Endpunkte zum Starten und Abfragen von Scanaufträgen.
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
	// für das spätere Web-Frontend bereit.
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Printf("HTTP-Server wurde beendet: %v", err)
	}
}
