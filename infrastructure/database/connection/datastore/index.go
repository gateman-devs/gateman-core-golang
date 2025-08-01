package datastore

var client *MongoClient

func ConnectToDatabase() {
	Connect(nil)
}

func CleanUp() {
	client.Disconnect()
}
