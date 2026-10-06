package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"network-scanner/backend/internal/model"
	"network-scanner/backend/internal/nmapxml"
	"network-scanner/backend/internal/repository"
)

const interruptedScanMessage = "Scan wurde abgebrochen, da die Scanner-Komponente neu gestartet wurde."

// ScanService koordiniert die Kommunikation mit dem Scanner
// und die dauerhafte Speicherung der Scanaufträge.
type ScanService struct {
	scanner    *ScannerService
	repository *repository.ScanRepository
}

// NewScanService erstellt den zentralen Service für Scanaufträge.
func NewScanService(
	scanner *ScannerService,
	repository *repository.ScanRepository,
) *ScanService {

	return &ScanService{
		scanner:    scanner,
		repository: repository,
	}
}

// CreateScan startet einen neuen Scanauftrag und speichert
// die erhaltene Scan-ID zusammen mit dem aktuellen Status.
func (s *ScanService) CreateScan(
	ctx context.Context,
	target string,
) (*model.ScanJob, error) {

	job, err := s.scanner.CreateScan(ctx, target)
	if err != nil {
		return nil, err
	}

	if err := s.repository.Create(ctx, job); err != nil {
		return nil, fmt.Errorf(
			"scan wurde gestartet, konnte aber nicht gespeichert werden: %w",
			err,
		)
	}

	return job, nil
}

// GetScan fragt den aktuellen Scanstatus ab und aktualisiert
// die gespeicherten Informationen in PostgreSQL.
func (s *ScanService) GetScan(
	ctx context.Context,
	id string,
) (*model.ScanJob, error) {

	job, err := s.scanner.GetScan(ctx, id)

	if err != nil {
		// Ist der Scanner erreichbar, kennt den Auftrag aber nicht mehr,
		// wird ein zuvor offener Scan kontrolliert als fehlgeschlagen markiert.
		if errors.Is(err, ErrScanNotFound) {
			return s.markScanAsFailed(
				ctx,
				id,
				interruptedScanMessage,
			)
		}

		// Bei einem vorübergehenden Kommunikationsproblem wird der
		// zuletzt bekannte Zustand aus PostgreSQL zurückgegeben.
		return s.repository.GetByScanID(ctx, id)
	}

	var details *nmapxml.ScanResult

	// Erst nach erfolgreichem Abschluss wird das XML verarbeitet.
	// Laufende oder fehlgeschlagene Scans besitzen noch kein
	// vollständiges Ergebnis, das ausgewertet werden könnte.
	if job.Status == "completed" {

		details, err = nmapxml.Parse([]byte(job.Output))
		if err != nil {
			return nil, fmt.Errorf(
				"scanergebnis konnte nicht ausgewertet werden: %w",
				err,
			)
		}
	}

	// Status und gegebenenfalls strukturierte Ergebnisse
	// werden gemeinsam über das Repository gespeichert.
	if err := s.repository.Update(ctx, job, details); err != nil {
		return nil, fmt.Errorf(
			"scanergebnis konnte nicht gespeichert werden: %w",
			err,
		)
	}

	return job, nil
}

// GetResults liest die bereits gespeicherten Scanergebnisse
// aus PostgreSQL und stellt sie für die REST-API bereit.
func (s *ScanService) GetResults(
	ctx context.Context,
	id string,
) (*model.ScanResults, error) {

	return s.repository.GetResults(ctx, id)
}

// GetHistory liest die zuletzt gespeicherten Scanläufe
// über das Repository aus PostgreSQL.
func (s *ScanService) GetHistory(
	ctx context.Context,
) (*model.ScanHistory, error) {

	return s.repository.GetHistory(ctx)
}

// SyncActiveScans gleicht offene Scanaufträge regelmässig
// mit der Scanner-Komponente ab.
func (s *ScanService) SyncActiveScans(ctx context.Context) error {

	scanIDs, err := s.repository.GetActiveScanIDs(ctx)
	if err != nil {
		return fmt.Errorf(
			"offene Scanaufträge konnten nicht geladen werden: %w",
			err,
		)
	}

	for _, id := range scanIDs {

		if err := ctx.Err(); err != nil {
			return err
		}

		job, err := s.scanner.GetScan(ctx, id)
		if err != nil {

			// Antwortet der Scanner mit 404, ist er erreichbar,
			// kennt den zuvor gespeicherten Auftrag aber nicht mehr.
			// Ein solcher Auftrag darf nicht dauerhaft auf pending
			// oder running stehen bleiben.
			if errors.Is(err, ErrScanNotFound) {

				failedJob, failErr := s.markScanAsFailed(
					ctx,
					id,
					interruptedScanMessage,
				)
				if failErr != nil {
					log.Printf(
						"Scan %s konnte nicht als fehlgeschlagen gespeichert werden: %v",
						id,
						failErr,
					)
					continue
				}

				log.Printf(
					"Scan %s automatisch auf %s gesetzt: Scanner kennt Auftrag nicht mehr",
					id,
					failedJob.Status,
				)

				continue
			}

			// Bei einem normalen Kommunikationsproblem bleibt der
			// aktuelle Datenbankstatus bestehen. Beim nächsten
			// Synchronisierungslauf wird erneut geprüft.
			log.Printf(
				"Scan %s konnte nicht synchronisiert werden: %v",
				id,
				err,
			)
			continue
		}

		var details *nmapxml.ScanResult

		if job.Status == "completed" {

			details, err = nmapxml.Parse([]byte(job.Output))
			if err != nil {
				log.Printf(
					"XML-Verarbeitung für Scan %s fehlgeschlagen: %v",
					id,
					err,
				)
				continue
			}
		}

		if err := s.repository.Update(ctx, job, details); err != nil {
			log.Printf(
				"Speicherung von Scan %s fehlgeschlagen: %v",
				id,
				err,
			)
			continue
		}

		if job.Status == "completed" || job.Status == "failed" {
			log.Printf(
				"Scan %s automatisch synchronisiert: %s",
				id,
				job.Status,
			)
		}
	}

	return nil
}

// markScanAsFailed übernimmt einen noch offenen Scanauftrag aus PostgreSQL
// und setzt ihn kontrolliert auf failed. Bereits abgeschlossene Aufträge
// werden nicht nachträglich verändert.
func (s *ScanService) markScanAsFailed(
	ctx context.Context,
	id string,
	message string,
) (*model.ScanJob, error) {

	job, err := s.repository.GetByScanID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf(
			"scanauftrag konnte nicht aus PostgreSQL geladen werden: %w",
			err,
		)
	}

	// Ein bereits endgültig abgeschlossener Datensatz darf nicht
	// durch einen späteren Synchronisierungslauf überschrieben werden.
	if job.Status == "completed" || job.Status == "failed" {
		return job, nil
	}

	job.Status = "failed"
	job.Error = message

	if err := s.repository.Update(ctx, job, nil); err != nil {
		return nil, fmt.Errorf(
			"fehlgeschlagener scanstatus konnte nicht gespeichert werden: %w",
			err,
		)
	}

	return job, nil
}
