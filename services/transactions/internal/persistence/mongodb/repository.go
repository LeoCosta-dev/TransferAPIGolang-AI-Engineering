package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lcosta/TransferAPIGolang/services/transactions/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/transactions/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Repository struct {
	client               *mongo.Client
	database             *mongo.Database
	balances, operations *mongo.Collection
}
type balanceDocument struct {
	ID        string        `bson:"_id"`
	Balance   int64         `bson:"balance"`
	Status    domain.Status `bson:"status"`
	UpdatedAt time.Time     `bson:"updated_at"`
}
type operationDocument struct {
	ID        string      `bson:"_id"`
	AccountID string      `bson:"account_id"`
	Type      domain.Type `bson:"type"`
	Amount    int64       `bson:"amount"`
	Balance   int64       `bson:"balance"`
	CreatedAt time.Time   `bson:"created_at"`
}

func Open(ctx context.Context, uri, database string) (*Repository, error) {
	if uri == "" || database == "" {
		return nil, errors.New("MONGODB_URI e MONGODB_DATABASE são obrigatórios")
	}
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("conectar ao MongoDB: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("verificar MongoDB: %w", err)
	}
	db := client.Database(database)
	r := &Repository{client: client, database: db, balances: db.Collection("account_balances"), operations: db.Collection("transactions")}
	return r, nil
}
func (r *Repository) Close(ctx context.Context) error { return r.client.Disconnect(ctx) }
func (r *Repository) Register(ctx context.Context, id string, status domain.Status) error {
	_, err := r.balances.UpdateOne(ctx, bson.D{{Key: "_id", Value: id}}, bson.D{{Key: "$setOnInsert", Value: balanceDocument{ID: id, Status: status, UpdatedAt: time.Now().UTC()}}}, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("registrar conta financeira: %w", err)
	}
	return nil
}
func (r *Repository) ChangeStatus(ctx context.Context, id string, from, to domain.Status) error {
	if !domain.CanTransition(from, to) {
		return domain.ErrInvalidStatusChange
	}
	session, err := r.client.StartSession()
	if err != nil {
		return fmt.Errorf("iniciar sessão MongoDB: %w", err)
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sc context.Context) (any, error) {
		filter := bson.D{{Key: "_id", Value: id}, {Key: "status", Value: from}}
		if to == domain.StatusClosed {
			filter = append(filter, bson.E{Key: "balance", Value: int64(0)})
		}
		result, err := r.balances.UpdateOne(sc, filter, bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: to}, {Key: "updated_at", Value: time.Now().UTC()}}}})
		if err != nil {
			return nil, fmt.Errorf("alterar status financeiro: %w", err)
		}
		if result.MatchedCount == 0 {
			var current balanceDocument
			err := r.balances.FindOne(sc, bson.D{{Key: "_id", Value: id}}).Decode(&current)
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, application.ErrAccountNotFound
			}
			if err != nil {
				return nil, err
			}
			if to == domain.StatusClosed && current.Balance != 0 {
				return nil, domain.ErrInsufficientBalance
			}
			return nil, domain.ErrInvalidStatusChange
		}
		return nil, nil
	})
	return err
}
func (r *Repository) Apply(ctx context.Context, accountID string, operation domain.Transaction) (domain.Transaction, error) {
	if operation.Type == "BALANCE" {
		var b balanceDocument
		err := r.balances.FindOne(ctx, bson.D{{Key: "_id", Value: accountID}}).Decode(&b)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.Transaction{}, application.ErrAccountNotFound
		}
		if err != nil {
			return domain.Transaction{}, fmt.Errorf("consultar saldo: %w", err)
		}
		return domain.Transaction{AccountID: accountID, Balance: b.Balance}, nil
	}
	session, err := r.client.StartSession()
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("iniciar sessão MongoDB: %w", err)
	}
	defer session.EndSession(ctx)
	var result domain.Transaction
	_, err = session.WithTransaction(ctx, func(sc context.Context) (any, error) {
		id := accountID + ":" + operation.IdempotencyKey
		var old operationDocument
		if err := r.operations.FindOne(sc, bson.D{{Key: "_id", Value: id}}).Decode(&old); err == nil {
			if old.Type != operation.Type || old.Amount != operation.Amount {
				return nil, application.ErrIdempotencyConflict
			}
			result = domain.Transaction{ID: old.ID, AccountID: old.AccountID, Type: old.Type, Amount: old.Amount, Balance: old.Balance, IdempotencyKey: operation.IdempotencyKey}
			return nil, nil
		} else if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("consultar idempotência: %w", err)
		}
		current := domain.Account{ID: accountID}
		var b balanceDocument
		if err := r.balances.FindOne(sc, bson.D{{Key: "_id", Value: accountID}}).Decode(&b); err == nil {
			current.Balance = b.Balance
			current.Status = b.Status
		} else if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("consultar saldo: %w", err)
		} else {
			return nil, application.ErrAccountNotFound
		}
		if operation.Type == domain.TypeCredit {
			err = current.Credit(operation.Amount)
		} else {
			err = current.Debit(operation.Amount)
		}
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		filter := bson.D{{Key: "_id", Value: accountID}, {Key: "status", Value: domain.StatusActive}}
		update := bson.D{{Key: "$set", Value: bson.D{{Key: "balance", Value: current.Balance}, {Key: "updated_at", Value: now}}}}
		resultUpdate, err := r.balances.UpdateOne(sc, filter, update)
		if err != nil {
			return nil, fmt.Errorf("atualizar saldo: %w", err)
		}
		if resultUpdate.MatchedCount == 0 {
			return nil, domain.ErrAccountBlocked
		}
		doc := operationDocument{ID: id, AccountID: accountID, Type: operation.Type, Amount: operation.Amount, Balance: current.Balance, CreatedAt: now}
		if _, err := r.operations.InsertOne(sc, doc); err != nil {
			return nil, fmt.Errorf("registrar movimentação: %w", err)
		}
		result = domain.Transaction{ID: id, AccountID: accountID, Type: operation.Type, Amount: operation.Amount, Balance: current.Balance, IdempotencyKey: operation.IdempotencyKey}
		return nil, nil
	})
	if err != nil {
		return domain.Transaction{}, err
	}
	return result, nil
}
