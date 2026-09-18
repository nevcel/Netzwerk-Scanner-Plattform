package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"network-scanner/scanner/internal/model"
	"network-scanner/scanner/internal/nmap"
)

// ScanService verwaltet die Scanaufträge und startet
// die eigentliche Nmap-Ausführung im Hintergrund.
type ScanService struct {
	executor *nmap.Executor

	// Die Scanaufträge werden in dieser ersten Ausbaustufe
	// vorübergehend im Arbeitsspeicher gespeichert.
	mu   sync.RWMutex
	jobs map[string]*model.ScanJob
}

// NewScanService erstellt einen neuen Service für Scanaufträge.
func NewScanService(executor *nmap.Executor) *ScanService {
	return &ScanService{
		executor: executor,
		jobs:     make(map[string]*model.ScanJob),
	}
}

// CreateScan prüft das Ziel und erstellt anschliessend
// einen neuen asynchronen Scanauftrag.
func (s *ScanService) CreateScan(target string) (*model.ScanJob, error) {
	if err := s.executor.ValidateTarget(target); err != nil {
		return nil, err
	}

	id, err := generateID()
	if err != nil {
		return nil, err
	}

	now := time.Now()

	job := &model.ScanJob{
		ID:        id,
		Target:    target,
		Status:    model.ScanStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	s.jobs[id] = job
	s.mu.Unlock()

	// Für die Rückgabe wird eine Kopie erstellt.
	// Dadurch bleibt die HTTP-Antwort unabhängig von den
	// anschliessenden Statusänderungen des Hintergrundprozesses.
	jobCopy := *job

	go s.runScan(id)

	return &jobCopy, nil
}

// GetScan liefert den aktuellen Zustand eines Scanauftrags.
func (s *ScanService) GetScan(id string) (*model.ScanJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]
	if !ok {
		return nil, errors.New("scanauftrag nicht gefunden")
	}

	// Es wird eine Kopie zurückgegeben, damit der interne
	// Zustand nicht ausserhalb des Services verändert werden kann.
	copy := *job
	return &copy, nil
}

// runScan führt den Scan aus und aktualisiert danach den Status.
func (s *ScanService) runScan(id string) {
	s.updateStatus(id, model.ScanStatusRunning, "", "")

	// Ein Timeout verhindert unbegrenzt laufende Scanprozesse.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	s.mu.RLock()
	target := s.jobs[id].Target
	s.mu.RUnlock()

	output, err := s.executor.Run(ctx, target)
	if err != nil {
		s.updateStatus(id, model.ScanStatusFailed, "", err.Error())
		return
	}

	s.updateStatus(id, model.ScanStatusCompleted, string(output), "")
}

// updateStatus aktualisiert einen Scanauftrag threadsicher.
func (s *ScanService) updateStatus(
	id string,
	status model.ScanStatus,
	output string,
	errMessage string,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return
	}

	job.Status = status
	job.Output = output
	job.Error = errMessage
	job.UpdatedAt = time.Now()
}

// generateID erzeugt eine zufällige ID für einen Scanauftrag.
func generateID() (string, error) {
	value := make([]byte, 8)

	if _, err := rand.Read(value); err != nil {
		return "", err
	}

	return hex.EncodeToString(value), nil
}
