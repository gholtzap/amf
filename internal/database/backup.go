package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gavin/amf/internal/logger"
	"go.mongodb.org/mongo-driver/bson"
)

type BackupManager struct {
	client *MongoDBClient
}

func NewBackupManager(client *MongoDBClient) *BackupManager {
	return &BackupManager{
		client: client,
	}
}

func (bm *BackupManager) BackupToDirectory(dirPath string) error {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupDir := filepath.Join(dirPath, fmt.Sprintf("backup_%s", timestamp))

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create timestamped backup directory: %w", err)
	}

	collections := []string{
		"ue_contexts",
		"n1n2_subscriptions",
		"event_subscriptions",
		"amf_status_subscriptions",
		"non_ue_n2_subscriptions",
	}

	for _, collName := range collections {
		if err := bm.backupCollection(collName, backupDir); err != nil {
			logger.DbLog.Errorf("Failed to backup collection %s: %v", collName, err)
			return err
		}
	}

	logger.DbLog.Infof("Backup completed successfully to %s", backupDir)
	return nil
}

func (bm *BackupManager) backupCollection(collectionName, backupDir string) error {
	collection := bm.client.GetCollection(collectionName)

	cursor, err := collection.Find(bm.client.Context(), bson.M{})
	if err != nil {
		return fmt.Errorf("failed to query collection %s: %w", collectionName, err)
	}
	defer cursor.Close(bm.client.Context())

	var documents []bson.M
	if err := cursor.All(bm.client.Context(), &documents); err != nil {
		return fmt.Errorf("failed to decode documents from %s: %w", collectionName, err)
	}

	filePath := filepath.Join(backupDir, fmt.Sprintf("%s.json", collectionName))
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create backup file for %s: %w", collectionName, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(documents); err != nil {
		return fmt.Errorf("failed to encode documents to JSON for %s: %w", collectionName, err)
	}

	logger.DbLog.Infof("Backed up %d documents from collection %s", len(documents), collectionName)
	return nil
}

func (bm *BackupManager) RestoreFromDirectory(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return fmt.Errorf("backup directory does not exist: %s", dirPath)
	}

	collections := []string{
		"ue_contexts",
		"n1n2_subscriptions",
		"event_subscriptions",
		"amf_status_subscriptions",
		"non_ue_n2_subscriptions",
	}

	for _, collName := range collections {
		filePath := filepath.Join(dirPath, fmt.Sprintf("%s.json", collName))

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			logger.DbLog.Warnf("Backup file not found for collection %s, skipping", collName)
			continue
		}

		if err := bm.restoreCollection(collName, filePath); err != nil {
			logger.DbLog.Errorf("Failed to restore collection %s: %v", collName, err)
			return err
		}
	}

	logger.DbLog.Infof("Restore completed successfully from %s", dirPath)
	return nil
}

func (bm *BackupManager) restoreCollection(collectionName, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open backup file for %s: %w", collectionName, err)
	}
	defer file.Close()

	var documents []bson.M
	decoder := json.NewDecoder(file)

	if err := decoder.Decode(&documents); err != nil {
		return fmt.Errorf("failed to decode JSON for %s: %w", collectionName, err)
	}

	if len(documents) == 0 {
		logger.DbLog.Infof("No documents to restore for collection %s", collectionName)
		return nil
	}

	collection := bm.client.GetCollection(collectionName)

	if _, err := collection.DeleteMany(bm.client.Context(), bson.M{}); err != nil {
		return fmt.Errorf("failed to clear collection %s before restore: %w", collectionName, err)
	}

	documentsInterface := make([]interface{}, len(documents))
	for i, doc := range documents {
		documentsInterface[i] = doc
	}

	if _, err := collection.InsertMany(bm.client.Context(), documentsInterface); err != nil {
		return fmt.Errorf("failed to insert documents into %s: %w", collectionName, err)
	}

	logger.DbLog.Infof("Restored %d documents to collection %s", len(documents), collectionName)
	return nil
}
