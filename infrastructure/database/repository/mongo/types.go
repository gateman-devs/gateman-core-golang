package mongo

import (
	"authone.usepolymer.co/infrastructure/database"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoRepository[T database.BaseModel] struct {
	Model *mongo.Collection
}

type FindOptions struct {
	Projection *interface{}
	Sort       *interface{}
	Skip       *int64
}
