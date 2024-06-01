package datastore

import (
	"context"
	"os"
	"time"

	"authone.usepolymer.co/infrastructure/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	OrgModel       *mongo.Collection
	OrgMemberModel *mongo.Collection
)

func connectMongo() *context.CancelFunc {
	url := os.Getenv("DB_URL")

	if url == "" {
		logger.Error("mongo url missing")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

	clientOpts := options.Client().ApplyURI(url)
	clientOpts.SetMinPoolSize(5)
	clientOpts.SetMaxPoolSize(10)

	client, err := mongo.Connect(ctx, clientOpts)

	if err != nil {
		logger.Warning("an error occured while starting the database", logger.LoggerOptions{Key: "error", Data: err})
		return &cancel
	}

	db := client.Database(os.Getenv("DB_NAME"))
	setUpIndexes(ctx, db)

	logger.Info("connected to mongodb successfully")
	return &cancel
}

// Set up the indexes for the database
func setUpIndexes(ctx context.Context, db *mongo.Database) {
	OrgModel = db.Collection("Organisation")

	OrgMemberModel = db.Collection("OrgModel")
	OrgMemberModel.Indexes().CreateMany(ctx, []mongo.IndexModel{{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index(),
	}})

	logger.Info("mongodb indexes set up successfully")
}
