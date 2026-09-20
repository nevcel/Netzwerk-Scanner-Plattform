package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"network-scanner/backend/internal/controller"
	"network-scanner/backend/internal/database"
	"network-scanner/backend/internal/repository"
	"network-scanner/backend/internal/service"
)

func main() {
	// Beim Start wird die Verbindung zu PostgreSQL überprüft.
	// Ohne funktionierende Datenbank wird der HTTP-Server nicht gestartet.
	dbPool, err := database.NewPostgresPool(context.Background())
	if err != nil {
		log.Printf("Datenbankfehler: %v", err)
		return
	}
	defer dbPool.Close()

	log.Println("Verbindung zu PostgreSQL erfolgreich hergestellt")

	// Die Scanner-Adresse wird über eine Umgebungsvariable konfiguriert.
	// Innerhalb von Docker Compose wird http://scanner:8081 verwendet.
	scannerURL := os.Getenv("SCANNER_URL")
	if scannerURL == "" {
		scannerURL = "http://localhost:8081"
	}

	// Der ScannerService übernimmt die HTTP-Kommunikation
	// mit der getrennten Scanner-Komponente.
	scannerService := service.NewScannerService(scannerURL)

	// Das Repository übernimmt die Datenbankzugriffe.
	scanRepository := repository.NewScanRepository(dbPool)

	// Der ScanService verbindet Scanner-Kommunikation
	// und dauerhafte Speicherung in PostgreSQL.
	scanService := service.NewScanService(
		scannerService,
		scanRepository,
	)

	// Offene Scanaufträge werden unabhängig vom Browser
	// regelmässig mit der Scanner-Komponente abgeglichen.
	syncCtx, stopSync := context.WithCancel(context.Background())
	defer stopSync()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			// Jede Synchronisierung erhält ein eigenes Timeout,
			// damit Kommunikationsprobleme den Prozess nicht blockieren.
			ctx, cancel := context.WithTimeout(
				syncCtx,
				30*time.Second,
			)

			err := scanService.SyncActiveScans(ctx)
			cancel()

			if err != nil && syncCtx.Err() == nil {
				log.Printf("Scan-Synchronisierung fehlgeschlagen: %v", err)
			}

			// Nach dem Abgleich wird auf die nächste Prüfung gewartet.
			select {
			case <-syncCtx.Done():
				return

			case <-ticker.C:
			}
		}
	}()

	// Die Controller verarbeiten die HTTP-Anfragen des Clients.
	scannerController := controller.NewScannerController(scannerService)
	scanController := controller.NewScanController(scanService)

	mux := http.NewServeMux()

	// Prüft die Erreichbarkeit des Backends.
	mux.HandleFunc(
		"GET /health",
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "backend ok")
		},
	)

	// Prüft die Verbindung zur Scanner-Komponente.
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
	// eines Scanlaufs aus PostgreSQL.
	mux.HandleFunc(
		"GET /api/scans/{id}/results",
		scanController.GetResults,
	)

	log.Println("Backend läuft auf Port 8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Printf("HTTP-Server wurde beendet: %v", err)
	}
}
