package mongodb

import (
	"context"
	"strings"
	"testing"
)

// Teste unitário do guard de configuração: não toca na rede porque Open
// valida os parâmetros antes de qualquer conexão.
func TestOpenRejectsMissingConfiguration(t *testing.T) {
	cases := []struct {
		name     string
		uri      string
		database string
		wantErr  string
	}{
		{"uri ausente", "", "transfer_api", "MONGODB_URI"},
		{"database ausente", "mongodb://localhost:27017", "", "MONGODB_DATABASE"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Open(context.Background(), tc.uri, tc.database)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("erro = %v, want contendo %q", err, tc.wantErr)
			}
		})
	}
}
