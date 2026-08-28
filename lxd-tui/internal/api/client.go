package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"lxd-tui/internal/models"
	"net/http"
	"time"
)

// APIClient membungkus komunikasi HTTP ke Go backend.
type APIClient struct {
	BaseURL string
	Token   string
	http    *http.Client
}

func NewAPIClient(baseURL, token string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		Token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *APIClient) FetchEnvInfo() (*models.EnvInfo, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/v1/environments/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tidak bisa menghubungi server: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server merespons status %d: %s", resp.StatusCode, string(body))
	}

	var info models.EnvInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("gagal membaca respons server: %w", err)
	}
	return &info, nil
}

type identifyRequest struct {
	Nama string `json:"nama"`
	NPM  string `json:"npm"`
}

// ErrIdentityMismatch dikembalikan saat environment sudah pernah diisi oleh
// praktikan lain, dan NPM yang di-submit sekarang tidak cocok. Ini SENGAJA
// bukan error fatal biasa — TUI menampilkan pesan yang jelas ke pengguna,
// bukan pesan error generik.
var ErrIdentityMismatch = errors.New("nama/npm tidak cocok dengan environment ini")

func (c *APIClient) SubmitIdentity(nama, npm string) error {
	payload, _ := json.Marshal(identifyRequest{Nama: nama, NPM: npm})

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/v1/environments/me/identify", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("tidak bisa menghubungi server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return ErrIdentityMismatch
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return errors.New("gagal mengirim data: " + string(body))
	}
	return nil
}
