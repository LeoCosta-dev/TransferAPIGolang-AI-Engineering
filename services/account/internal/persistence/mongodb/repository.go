package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lcosta/TransferAPIGolang/services/account/internal/application"
	"github.com/lcosta/TransferAPIGolang/services/account/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Repository struct {
	client     *mongo.Client
	collection *mongo.Collection
}
type document struct {
	ID        string        `bson:"_id"`
	Name      string        `bson:"name"`
	Document  string        `bson:"document"`
	Status    domain.Status `bson:"status"`
	CreatedAt time.Time     `bson:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at"`
}

func Open(ctx context.Context, uri, database string) (*Repository, error) {
	if uri == "" || database == "" {
		return nil, fmt.Errorf("MONGODB_URI e MONGODB_DATABASE são obrigatórios")
	}
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("conectar ao MongoDB: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("verificar MongoDB: %w", err)
	}
	collection := client.Database(database).Collection("accounts")
	if _, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "document", Value: 1}}, Options: options.Index().SetUnique(true)}); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("criar índice de documento: %w", err)
	}
	return &Repository{client: client, collection: collection}, nil
}

func (repository *Repository) Close(ctx context.Context) error {
	return repository.client.Disconnect(ctx)
}

func (repository *Repository) Create(ctx context.Context, account domain.Account) error {
	_, err := repository.collection.InsertOne(ctx, document{ID: account.ID, Name: account.Name, Document: account.Document, Status: account.Status, CreatedAt: account.CreatedAt, UpdatedAt: account.UpdatedAt})
	if mongo.IsDuplicateKeyError(err) {
		return application.ErrDuplicateDocument
	}
	if err != nil {
		return fmt.Errorf("criar conta: %w", err)
	}
	return nil
}

func (repository *Repository) FindByID(ctx context.Context, id string) (domain.Account, error) {
	var value document
	if err := repository.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&value); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.Account{}, application.ErrAccountNotFound
		}
		return domain.Account{}, fmt.Errorf("buscar conta: %w", err)
	}
	return domain.Account{ID: value.ID, Name: value.Name, Document: value.Document, Status: value.Status, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}, nil
}

func (repository *Repository) Update(ctx context.Context, account domain.Account) error {
	result, err := repository.collection.UpdateOne(ctx, bson.D{{Key: "_id", Value: account.ID}}, bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: account.Name}, {Key: "status", Value: account.Status}, {Key: "updated_at", Value: account.UpdatedAt}}}})
	if err != nil {
		return fmt.Errorf("atualizar conta: %w", err)
	}
	if result.MatchedCount == 0 {
		return application.ErrAccountNotFound
	}
	return nil
}
