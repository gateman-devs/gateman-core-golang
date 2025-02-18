package database

import "gateman.io/infrastructure/database/connection"

func SetUpDatabase() {
	connection.ConnectToDatabase()
}

type BaseModel interface {
	ParseModel() any
}
