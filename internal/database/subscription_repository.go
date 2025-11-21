package database

import (
	"context"
	"fmt"
	"time"

	"github.com/gavin/amf/internal/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SubscriptionRepository struct {
	n1n2Collection         *mongo.Collection
	eventCollection        *mongo.Collection
	amfStatusCollection    *mongo.Collection
	nonUeN2Collection      *mongo.Collection
	ctx                    context.Context
}

type N1N2SubscriptionDocument struct {
	SubscriptionId      string    `bson:"subscription_id"`
	UeContextId         string    `bson:"ue_context_id"`
	N1MessageClass      string    `bson:"n1_message_class,omitempty"`
	N1NotifyCallbackUri string    `bson:"n1_notify_callback_uri,omitempty"`
	N2InformationClass  string    `bson:"n2_information_class,omitempty"`
	N2NotifyCallbackUri string    `bson:"n2_notify_callback_uri,omitempty"`
	NfId                string    `bson:"nf_id,omitempty"`
	UpdatedAt           time.Time `bson:"updated_at"`
}

type GenericSubscriptionDocument struct {
	SubscriptionId string                 `bson:"subscription_id"`
	Data           map[string]interface{} `bson:"data"`
	UpdatedAt      time.Time              `bson:"updated_at"`
}

func NewSubscriptionRepository(client *MongoDBClient) *SubscriptionRepository {
	n1n2Collection := client.GetCollection("n1n2_subscriptions")
	eventCollection := client.GetCollection("event_subscriptions")
	amfStatusCollection := client.GetCollection("amf_status_subscriptions")
	nonUeN2Collection := client.GetCollection("non_ue_n2_subscriptions")

	ctx := client.Context()

	createIndex := func(collection *mongo.Collection) {
		indexModel := mongo.IndexModel{
			Keys:    bson.D{{Key: "subscription_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		}
		_, err := collection.Indexes().CreateOne(ctx, indexModel)
		if err != nil {
			logger.InitLog.Warnf("Failed to create index on %s: %v", collection.Name(), err)
		}
	}

	createIndex(n1n2Collection)
	createIndex(eventCollection)
	createIndex(amfStatusCollection)
	createIndex(nonUeN2Collection)

	logger.InitLog.Info("Subscription Repository initialized")

	return &SubscriptionRepository{
		n1n2Collection:      n1n2Collection,
		eventCollection:     eventCollection,
		amfStatusCollection: amfStatusCollection,
		nonUeN2Collection:   nonUeN2Collection,
		ctx:                 ctx,
	}
}

func (r *SubscriptionRepository) SaveN1N2Subscription(subscriptionId, ueContextId, n1MessageClass, n1NotifyCallbackUri, n2InformationClass, n2NotifyCallbackUri, nfId string) error {
	doc := N1N2SubscriptionDocument{
		SubscriptionId:      subscriptionId,
		UeContextId:         ueContextId,
		N1MessageClass:      n1MessageClass,
		N1NotifyCallbackUri: n1NotifyCallbackUri,
		N2InformationClass:  n2InformationClass,
		N2NotifyCallbackUri: n2NotifyCallbackUri,
		NfId:                nfId,
		UpdatedAt:           time.Now(),
	}

	filter := bson.M{"subscription_id": subscriptionId}
	opts := options.Replace().SetUpsert(true)

	_, err := r.n1n2Collection.ReplaceOne(r.ctx, filter, doc, opts)
	if err != nil {
		return fmt.Errorf("failed to save N1N2 subscription: %w", err)
	}

	logger.DbLog.Debugf("N1N2 subscription saved to MongoDB: %s", subscriptionId)
	return nil
}

func (r *SubscriptionRepository) FindN1N2Subscription(subscriptionId string) (interface{}, error) {
	var doc N1N2SubscriptionDocument

	filter := bson.M{"subscription_id": subscriptionId}
	err := r.n1n2Collection.FindOne(r.ctx, filter).Decode(&doc)

	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find N1N2 subscription: %w", err)
	}

	return &doc, nil
}

func (r *SubscriptionRepository) FindAllN1N2Subscriptions() ([]interface{}, error) {
	cursor, err := r.n1n2Collection.Find(r.ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find N1N2 subscriptions: %w", err)
	}
	defer cursor.Close(r.ctx)

	var subscriptions []interface{}
	for cursor.Next(r.ctx) {
		var doc N1N2SubscriptionDocument
		if err := cursor.Decode(&doc); err != nil {
			logger.DbLog.Warnf("Failed to decode N1N2 subscription: %v", err)
			continue
		}
		subscriptions = append(subscriptions, &doc)
	}

	logger.DbLog.Infof("Loaded %d N1N2 subscriptions from MongoDB", len(subscriptions))
	return subscriptions, nil
}

func (r *SubscriptionRepository) DeleteN1N2Subscription(subscriptionId string) error {
	filter := bson.M{"subscription_id": subscriptionId}

	_, err := r.n1n2Collection.DeleteOne(r.ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete N1N2 subscription: %w", err)
	}

	logger.DbLog.Debugf("N1N2 subscription deleted from MongoDB: %s", subscriptionId)
	return nil
}

func (r *SubscriptionRepository) SaveEventSubscription(subscriptionId string, data map[string]interface{}) error {
	doc := GenericSubscriptionDocument{
		SubscriptionId: subscriptionId,
		Data:           data,
		UpdatedAt:      time.Now(),
	}

	filter := bson.M{"subscription_id": subscriptionId}
	opts := options.Replace().SetUpsert(true)

	_, err := r.eventCollection.ReplaceOne(r.ctx, filter, doc, opts)
	if err != nil {
		return fmt.Errorf("failed to save event subscription: %w", err)
	}

	logger.DbLog.Debugf("Event subscription saved to MongoDB: %s", subscriptionId)
	return nil
}

func (r *SubscriptionRepository) FindAllEventSubscriptions() ([]interface{}, error) {
	cursor, err := r.eventCollection.Find(r.ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find event subscriptions: %w", err)
	}
	defer cursor.Close(r.ctx)

	var subscriptions []interface{}
	for cursor.Next(r.ctx) {
		var doc GenericSubscriptionDocument
		if err := cursor.Decode(&doc); err != nil {
			logger.DbLog.Warnf("Failed to decode event subscription: %v", err)
			continue
		}
		subscriptions = append(subscriptions, &doc)
	}

	logger.DbLog.Infof("Loaded %d event subscriptions from MongoDB", len(subscriptions))
	return subscriptions, nil
}

func (r *SubscriptionRepository) DeleteEventSubscription(subscriptionId string) error {
	filter := bson.M{"subscription_id": subscriptionId}

	_, err := r.eventCollection.DeleteOne(r.ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete event subscription: %w", err)
	}

	logger.DbLog.Debugf("Event subscription deleted from MongoDB: %s", subscriptionId)
	return nil
}

func (r *SubscriptionRepository) SaveAMFStatusSubscription(subscriptionId string, data map[string]interface{}) error {
	doc := GenericSubscriptionDocument{
		SubscriptionId: subscriptionId,
		Data:           data,
		UpdatedAt:      time.Now(),
	}

	filter := bson.M{"subscription_id": subscriptionId}
	opts := options.Replace().SetUpsert(true)

	_, err := r.amfStatusCollection.ReplaceOne(r.ctx, filter, doc, opts)
	if err != nil {
		return fmt.Errorf("failed to save AMF status subscription: %w", err)
	}

	logger.DbLog.Debugf("AMF status subscription saved to MongoDB: %s", subscriptionId)
	return nil
}

func (r *SubscriptionRepository) FindAllAMFStatusSubscriptions() ([]interface{}, error) {
	cursor, err := r.amfStatusCollection.Find(r.ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find AMF status subscriptions: %w", err)
	}
	defer cursor.Close(r.ctx)

	var subscriptions []interface{}
	for cursor.Next(r.ctx) {
		var doc GenericSubscriptionDocument
		if err := cursor.Decode(&doc); err != nil {
			logger.DbLog.Warnf("Failed to decode AMF status subscription: %v", err)
			continue
		}
		subscriptions = append(subscriptions, &doc)
	}

	logger.DbLog.Infof("Loaded %d AMF status subscriptions from MongoDB", len(subscriptions))
	return subscriptions, nil
}

func (r *SubscriptionRepository) DeleteAMFStatusSubscription(subscriptionId string) error {
	filter := bson.M{"subscription_id": subscriptionId}

	_, err := r.amfStatusCollection.DeleteOne(r.ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete AMF status subscription: %w", err)
	}

	logger.DbLog.Debugf("AMF status subscription deleted from MongoDB: %s", subscriptionId)
	return nil
}
