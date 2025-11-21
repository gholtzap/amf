package database

import (
	"context"
	"fmt"
	"time"

	"github.com/gavin/amf/internal/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBClient struct {
	client   *mongo.Client
	database *mongo.Database
	ctx      context.Context
}

func NewMongoDBClient(uri, dbName string) (*MongoDBClient, error) {
	ctx := context.Background()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(dbName)
	logger.InitLog.Infof("Connected to MongoDB database: %s", dbName)

	return &MongoDBClient{
		client:   client,
		database: database,
		ctx:      ctx,
	}, nil
}

func (m *MongoDBClient) GetCollection(name string) *mongo.Collection {
	return m.database.Collection(name)
}

func (m *MongoDBClient) Disconnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := m.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
	}

	logger.InitLog.Info("Disconnected from MongoDB")
	return nil
}

func (m *MongoDBClient) Context() context.Context {
	return m.ctx
}
