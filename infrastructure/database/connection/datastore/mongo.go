package datastore

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"gateman.io/infrastructure/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	WorkspaceModel          *mongo.Collection
	WorkspaceMemberModel    *mongo.Collection
	ApplicationModel        *mongo.Collection
	AppUserModel            *mongo.Collection
	UserModel               *mongo.Collection
	SubscriptionPlanModel   *mongo.Collection
	ActiveSubscriptionModel *mongo.Collection
	WorkspaceInviteModel    *mongo.Collection
	TransactionModel        *mongo.Collection
	KYCIdentityDataModel    *mongo.Collection
	HelpCenterModel         *mongo.Collection
	RequestActivityLogModel *mongo.Collection
)

type MongoClient struct {
	Client   *mongo.Client
	Database *mongo.Database
	ctx      context.Context
	cancel   context.CancelFunc
}

var (
	mongoInstance *MongoClient
	mongoOnce     sync.Once
)

type Config struct {
	URI                    string
	DatabaseName           string
	MaxPoolSize            uint64
	MinPoolSize            uint64
	MaxConnIdleTime        time.Duration
	MaxConnecting          uint64
	ConnectTimeout         time.Duration
	ServerSelectionTimeout time.Duration
	SocketTimeout          time.Duration
	HeartbeatInterval      time.Duration
}

func GetDefaultConfigOpts() *Config {
	return &Config{
		URI:                    os.Getenv("DB_URL"),
		DatabaseName:           os.Getenv("DB_NAME"),
		MaxPoolSize:            100,
		MinPoolSize:            5,
		MaxConnIdleTime:        30 * time.Minute,
		MaxConnecting:          10,
		ConnectTimeout:         10 * time.Second,
		ServerSelectionTimeout: 5 * time.Second,
		SocketTimeout:          30 * time.Second,
		HeartbeatInterval:      10 * time.Second,
	}
}

func Connect(config *Config) *MongoClient {
	if config == nil {
		config = GetDefaultConfigOpts()
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)

	clientOptions := options.Client().
		ApplyURI(config.URI).
		SetMaxPoolSize(config.MaxPoolSize).
		SetMinPoolSize(config.MinPoolSize).
		SetMaxConnIdleTime(config.MaxConnIdleTime).
		SetMaxConnecting(config.MaxConnecting).
		SetConnectTimeout(config.ConnectTimeout).
		SetServerSelectionTimeout(config.ServerSelectionTimeout).
		SetSocketTimeout(config.SocketTimeout).
		SetHeartbeatInterval(config.HeartbeatInterval).
		SetRetryWrites(true).
		SetRetryReads(true)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		cancel()
		panic(fmt.Errorf("failed to connect to MongoDB: %w", err))
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		cancel()
		panic(fmt.Errorf("failed to ping MongoDB: %w", err))
	}

	database := client.Database(config.DatabaseName)

	mongoClient := &MongoClient{
		Client:   client,
		Database: database,
		ctx:      ctx,
		cancel:   cancel,
	}

	log.Printf("Successfully connected to MongoDB database: %s", config.DatabaseName)
	setUpIndexes(context.TODO(), mongoClient.Database)
	return mongoClient
}

func GetInstance() (*MongoClient, error) {
	var err error
	mongoOnce.Do(func() {
		mongoInstance = Connect(nil)
	})
	return mongoInstance, err
}

func (mc *MongoClient) Disconnect() error {
	if mc.cancel != nil {
		mc.cancel()
	}

	if mc.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := mc.Client.Disconnect(ctx); err != nil {
			return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
		}
		log.Println("Successfully disconnected from MongoDB")
	}
	return nil
}

func (mc *MongoClient) GetCollection(name string) *mongo.Collection {
	return mc.Database.Collection(name)
}

func setUpIndexes(ctx context.Context, db *mongo.Database) {
	WorkspaceModel = db.Collection("Workspaces")

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

	AppUserModel = db.Collection("AppUsers")
	AppUserModel.Indexes().CreateMany(ctx, []mongo.IndexModel{{
		Keys:    bson.D{{Key: "appID", Value: 1}},
		Options: options.Index(),
	}, {
		Keys:    bson.D{{Key: "userID", Value: 1}},
		Options: options.Index(),
	}})

	ActiveSubscriptionModel = db.Collection("ActiveSubscriptions")
	ActiveSubscriptionModel.Indexes().CreateMany(ctx, []mongo.IndexModel{{
		Keys:    bson.D{{Key: "appID", Value: 1}},
		Options: options.Index(),
	}})

	KYCIdentityDataModel = db.Collection("KYCIdentityData")
	KYCIdentityDataModel.Indexes().CreateMany(ctx, []mongo.IndexModel{{
		Keys:    bson.D{{Key: "userID", Value: 1}},
		Options: options.Index(),
	}})

	TransactionModel = db.Collection("Transactions")
	TransactionModel.Indexes().CreateMany(ctx, []mongo.IndexModel{{
		Keys:    bson.D{{Key: "appID", Value: 1}},
		Options: options.Index(),
	}, {
		Keys:    bson.D{{Key: "refID", Value: 1}},
		Options: options.Index(),
	}, {
		Keys:    bson.D{{Key: "workspaceID", Value: 1}},
		Options: options.Index(),
	}})

	SubscriptionPlanModel = db.Collection("SubscriptionPlans")

	HelpCenterModel = db.Collection("HelpCenter")
	HelpCenterModel.Indexes().CreateMany(ctx, []mongo.IndexModel{{
		Keys:    bson.D{{Key: "workspace", Value: 1}},
		Options: options.Index(),
	}, {
		Keys:    bson.D{{Key: "member", Value: 1}},
		Options: options.Index(),
	}, {
		Keys:    bson.D{{Key: "status", Value: 1}},
		Options: options.Index(),
	}})

	RequestActivityLogModel = db.Collection("RequestActivityLogs")
	RequestActivityLogModel.Indexes().CreateMany(ctx, []mongo.IndexModel{{
		Keys:    bson.D{{Key: "appID", Value: 1}},
		Options: options.Index(),
	}, {
		Keys:    bson.D{{Key: "timestamp", Value: -1}},
		Options: options.Index(),
	}, {
		Keys:    bson.D{{Key: "ipAddress", Value: 1}},
		Options: options.Index(),
	}})

	logger.Info("mongodb indexes set up successfully")
}
