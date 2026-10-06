package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"sync"
	"time"

	"network-scanner/scanner/internal/model"
	"network-scanner/scanner/internal/nmap"
)

const (
	// Maximal zwei Nmap-Scans dürfen gleichzeitig ausgeführt werden.
	maxConcurrentScans = 2

	// Die Warteschlange nimmt maximal 100 zusätzliche Aufträge auf.
	maxQueuedScans = 100
)

var (
	// ErrQueueFull erlaubt dem HTTP-Controller, eine volle
	// Warteschlange von einem ungültigen Scanziel zu unterscheiden.
	ErrQueueFull = errors.New("scan-warteschlange ist voll")

	// ErrScanNotFound wird bei einer unbekannten Scan-ID verwendet.
	ErrScanNotFound = errors.New("scanauftrag nicht gefunden")
)

// ScanService verwaltet die Scanaufträge und koordiniert
// deren Ausführung über eine begrenzte Anzahl von Workern.
type ScanService struct {
	executor *nmap.Executor

	mu   sync.RWMutex
	jobs map[string]*model.ScanJob

	// Wartende Scanaufträge werden über diesen Channel
	// an die verfügbaren Worker weitergegeben.
	queue chan string
}

// NewScanService erstellt den ScanService und startet
// die Worker für die asynchrone Verarbeitung.
func NewScanService(executor *nmap.Executor) *ScanService {
	s := &ScanService{
		executor: executor,
		jobs:     make(map[string]*model.ScanJob),
		queue:    make(chan string, maxQueuedScans),
	}

	// Jeder Worker verarbeitet immer nur einen Scan gleichzeitig.
	for i := 0; i < maxConcurrentScans; i++ {
		go s.worker()
	}

	return s
}

// CreateScan validiert das Ziel und erstellt einen neuen Scanauftrag.
// Die Ausführung erfolgt erst, sobald ein Worker verfügbar ist.
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

	// Die Kopie verhindert, dass eine spätere Statusänderung
	// gleichzeitig die HTTP-Antwort verändert.
	jobCopy := *job

	s.mu.Lock()
	s.jobs[id] = job
	s.mu.Unlock()

	// Der Auftrag wird ohne längere Wartezeit angenommen.
	// Ist die Warteschlange voll, wird er kontrolliert abgelehnt.
	select {
	case s.queue <- id:
		return &jobCopy, nil

	default:
		s.mu.Lock()
		delete(s.jobs, id)
		s.mu.Unlock()

		return nil, ErrQueueFull
	}
}

// GetScan liefert eine Kopie des aktuellen Scanauftrags.
func (s *ScanService) GetScan(id string) (*model.ScanJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]
	if !ok {
		return nil, ErrScanNotFound
	}

	jobCopy := *job

	return &jobCopy, nil
}

// worker wartet auf Scanaufträge und verarbeitet diese nacheinander.
// Da zwei Worker gestartet werden, laufen maximal zwei Scans parallel.
func (s *ScanService) worker() {
	for id := range s.queue {
		s.runScan(id)
	}
}

// runScan führt den zugewiesenen Scan aus und aktualisiert
// anschliessend den Status und das Ergebnis.
func (s *ScanService) runScan(id string) {
	s.mu.RLock()

	job, ok := s.jobs[id]
	if !ok {
		s.mu.RUnlock()
		return
	}

	target := job.Target

	s.mu.RUnlock()

	// Der Status wechselt erst auf running,
	// wenn ein Worker tatsächlich mit der Ausführung beginnt.
	s.updateStatus(id, model.ScanStatusRunning, "", "")

	// Ein Timeout begrenzt die maximale Laufzeit von Nmap.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Minute,
	)
	defer cancel()

	output, err := s.executor.Run(ctx, target)

	if err != nil {
		log.Printf(
			"Scan %s für Ziel %s fehlgeschlagen: %v",
			id,
			target,
			err,
		)

		s.updateStatus(
			id,
			model.ScanStatusFailed,
			"",
			err.Error(),
		)
		return
	}

	s.updateStatus(
		id,
		model.ScanStatusCompleted,
		string(output),
		"",
	)
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

// generateID erzeugt eine zufällige ID für jeden Scanauftrag.
func generateID() (string, error) {
	value := make([]byte, 8)

	if _, err := rand.Read(value); err != nil {
		return "", err
	}

	return hex.EncodeToString(value), nil
}
