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
	WorkspaceModel        *mongo.Collection
	WorkspaceMemberModel  *mongo.Collection
	ApplicationModel      *mongo.Collection
	UserModel             *mongo.Collection
	SubscriptionPlanModel *mongo.Collection
	WorkspaceInviteModel  *mongo.Collection
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
	WorkspaceModel = db.Collection("Workspaces")
	WorkspaceModel.Indexes().CreateMany(ctx, []mongo.IndexModel{{
		Keys:    bson.D{{Key: "createdBy", Value: 1}},
		Options: options.Index(),
	}})

	WorkspaceMemberModel = db.Collection("WorkspaceMembers")
	WorkspaceMemberModel.Indexes().CreateMany(ctx, []mongo.IndexModel{{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index(),
	}, {
		Keys:    bson.D{{Key: "workspaceID", Value: 1}},
		Options: options.Index(),
	}, {
		Keys:    bson.D{{Key: "userID", Value: 1}},
		Options: options.Index(),
	}})

	UserModel = db.Collection("Users")
	UserModel.Indexes().CreateMany(ctx, []mongo.IndexModel{{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index(),
	}})

	ApplicationModel = db.Collection("Applications")
	ApplicationModel.Indexes().CreateMany(ctx, []mongo.IndexModel{{
		Keys:    bson.D{{Key: "workspaceID", Value: 1}},
		Options: options.Index(),
	}, {
		Keys:    bson.D{{Key: "appID", Value: 1}},
		Options: options.Index(),
	}})

	WorkspaceInviteModel = db.Collection("WorkspaceInvites")
	WorkspaceInviteModel.Indexes().CreateMany(ctx, []mongo.IndexModel{{
		Keys:    bson.D{{Key: "workspaceID", Value: 1}},
		Options: options.Index(),
	}})

	SubscriptionPlanModel = db.Collection("SubscriptionPlans")

	logger.Info("mongodb indexes set up successfully")
}
