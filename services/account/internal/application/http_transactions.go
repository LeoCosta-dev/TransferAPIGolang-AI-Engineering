package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
)

type HTTPFinancialGateway struct {
	baseURL string
	client  *http.Client
}

func NewHTTPFinancialGateway(baseURL string, client *http.Client) (*HTTPFinancialGateway, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("TRANSACTIONS_SERVICE_URL é obrigatório")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &HTTPFinancialGateway{strings.TrimRight(baseURL, "/"), client}, nil
}
func (g *HTTPFinancialGateway) Register(ctx context.Context, id string, status domain.Status) error {
	return g.send(ctx, "/internal/v1/accounts/"+id+"/register", status)
}
func (g *HTTPFinancialGateway) ChangeStatus(ctx context.Context, id string, from, to domain.Status) error {
	body, _ := json.Marshal(struct {
		From   domain.Status `json:"from"`
		Status domain.Status `json:"status"`
	}{from, to})
	return g.request(ctx, "/internal/v1/accounts/"+id+"/status", body)
}
func (g *HTTPFinancialGateway) send(ctx context.Context, path string, status domain.Status) error {
	body, _ := json.Marshal(struct {
		Status domain.Status `json:"status"`
	}{status})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("sincronizar estado financeiro: %w", err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Printf("erro ao fechar corpo da resposta: %v", err)
		}
	}()
	if res.StatusCode == http.StatusConflict {
		return ErrAccountHasBalance
	}
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return fmt.Errorf("sincronizar estado financeiro: status %d", res.StatusCode)
	}
	return nil
}
func (g *HTTPFinancialGateway) request(ctx context.Context, path string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("sincronizar estado financeiro: %w", err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Printf("erro ao fechar corpo da resposta: %v", err)
		}
	}()
	if res.StatusCode == http.StatusConflict {
		return ErrAccountHasBalance
	}
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return fmt.Errorf("sincronizar estado financeiro: status %d", res.StatusCode)
	}
	return nil
}
