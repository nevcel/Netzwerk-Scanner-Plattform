package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"network-scanner/backend/internal/model"
)

var (
	ErrScanNotFound = errors.New(
		"scanauftrag in scanner-komponente nicht gefunden",
	)

	ErrInvalidTarget = errors.New(
		"ungültiges scanziel",
	)

	ErrScannerBusy = errors.New(
		"scanner-komponente ist ausgelastet",
	)

	ErrScannerUnavailable = errors.New(
		"scanner-komponente ist nicht erreichbar",
	)
)

// ScannerService übernimmt die Kommunikation zwischen dem Backend
// und der getrennten Scanner-Komponente.
type ScannerService struct {
	baseURL string
	client  *http.Client
}

func NewScannerService(baseURL string) *ScannerService {
	return &ScannerService{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

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
		return fmt.Errorf("%w: %v", ErrScannerUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"%w: status %d",
			ErrScannerUnavailable,
			resp.StatusCode,
		)
	}

	return nil
}

func (s *ScannerService) CreateScan(
	ctx context.Context,
	target string,
) (*model.ScanJob, error) {

	if err := ValidateTarget(target); err != nil {
		return nil, fmt.Errorf(
			"%w: %v",
			ErrInvalidTarget,
			err,
		)
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
		return nil, fmt.Errorf(
			"%w: %v",
			ErrScannerUnavailable,
			err,
		)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusAccepted:
		var job model.ScanJob

		if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
			return nil, err
		}

		return &job, nil

	case http.StatusServiceUnavailable:
		return nil, ErrScannerBusy

	case http.StatusBadRequest:
		return nil, ErrInvalidTarget

	default:
		return nil, fmt.Errorf(
			"unerwartete antwort der scanner-komponente: status %d",
			resp.StatusCode,
		)
	}
}

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
		return nil, fmt.Errorf(
			"%w: %v",
			ErrScannerUnavailable,
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrScanNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"scanauftrag konnte nicht geladen werden: status %d",
			resp.StatusCode,
		)
	}

	var job model.ScanJob

	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, err
	}

	return &job, nil
}
