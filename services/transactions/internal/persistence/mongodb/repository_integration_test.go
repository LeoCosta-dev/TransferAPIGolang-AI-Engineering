package mongodb

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/lcosta/TransferAPIGolang/services/transactions/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/transactions/internal/domain"
)

func newIntegrationRepository(t *testing.T) *Repository {
	t.Helper()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("teste de integração requer MONGODB_URI apontando para MongoDB com suporte a transações")
	}
	r, err := Open(context.Background(), uri, "transactions_integration_"+time.Now().UTC().Format("20060102150405.000000000"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.database.Drop(context.Background()); _ = r.Close(context.Background()) })
	return r
}
func TestMongoFinancialConsistency(t *testing.T) {
	r := newIntegrationRepository(t)
	ctx := context.Background()
	id := "account-1"
	if err := r.Register(ctx, id, domain.StatusActive); err != nil {
		t.Fatal(err)
	}
	credit := domain.Transaction{AccountID: id, Type: domain.TypeCredit, Amount: 100, IdempotencyKey: "credit"}
	if _, err := r.Apply(ctx, id, credit); err != nil {
		t.Fatal(err)
	}
	replay, err := r.Apply(ctx, id, credit)
	if err != nil || replay.Balance != 100 {
		t.Fatalf("%+v %v", replay, err)
	}
	if _, err := r.Apply(ctx, id, domain.Transaction{AccountID: id, Type: domain.TypeDebit, Amount: 1, IdempotencyKey: "credit"}); !errors.Is(err, application.ErrIdempotencyConflict) {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := r.Apply(ctx, id, domain.Transaction{AccountID: id, Type: domain.TypeDebit, Amount: 15, IdempotencyKey: "d" + string(rune('a'+i))})
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil && !errors.Is(err, domain.ErrInsufficientBalance) {
			t.Fatal(err)
		}
	}
	b, err := r.Apply(ctx, id, domain.Transaction{Type: "BALANCE"})
	if err != nil || b.Balance < 0 {
		t.Fatalf("%+v %v", b, err)
	}
	if err := r.ChangeStatus(ctx, id, domain.StatusActive, domain.StatusClosed); !errors.Is(err, domain.ErrInsufficientBalance) {
		t.Fatal(err)
	}
}
