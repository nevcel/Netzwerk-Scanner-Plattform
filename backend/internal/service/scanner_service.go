package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"network-scanner/backend/internal/model"
)

// ScannerService übernimmt die Kommunikation zwischen dem Backend
// und der getrennten Scanner-Komponente.
type ScannerService struct {
	baseURL string
	client  *http.Client
}

// NewScannerService erstellt einen neuen Scanner-Service.
// Ein Timeout verhindert, dass das Backend bei Kommunikationsproblemen
// unbegrenzt auf die Scanner-Komponente wartet.
func NewScannerService(baseURL string) *ScannerService {
	return &ScannerService{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Health prüft, ob die Scanner-Komponente erreichbar ist.
func (s *ScannerService) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		s.baseURL+"/health",
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("scanner returned status %d", resp.StatusCode)
	}

	return nil
}

// CreateScan übermittelt einen neuen Scanauftrag an die Scanner-Komponente.
func (s *ScannerService) CreateScan(
	ctx context.Context,
	target string,
) (*model.ScanJob, error) {

	if err := ValidateTarget(target); err != nil {
		return nil, err
	}

	body, err := json.Marshal(model.ScanRequest{
		Target: target,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.baseURL+"/scan",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("scanner lehnte den scanauftrag ab")
	}

	var job model.ScanJob

	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, err
	}

	return &job, nil
}

// GetScan fragt den aktuellen Zustand eines Scanauftrags
// bei der Scanner-Komponente ab.
func (s *ScannerService) GetScan(
	ctx context.Context,
	id string,
) (*model.ScanJob, error) {

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		s.baseURL+"/scan/"+id,
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scanauftrag konnte nicht geladen werden")
	}

	var job model.ScanJob

	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, err
	}

	return &job, nil
}
