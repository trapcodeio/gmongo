package gmongo

import (
	"context"
	"testing"
)

func testConnectToDb() *Client {
	// connect to db
	client, err := ConnectUsingString("mongodb://localhost:27017", "gmongo")
	if err != nil {
		panic(err)
	}

	return client
}

// Test Connection via Connection String
func Test_ConnectUsingString(t *testing.T) {
	// arrange
	connectionString := "mongodb://localhost:27017"
	database := "gmongo"

	// act
	client, err := ConnectUsingString(connectionString, database)
	if err != nil {
		t.Error(err)
	}

	// check if connected
	if !client.IsConnected() {
		t.Error("Not connected")
	}

	// close connection
	err = client.MongoClient.Disconnect(context.TODO())
	if err != nil {
		t.Error(err)
	}
}

func Test_ConnectUsingCredentials(t *testing.T) {
	// arrange
	credentials := &ConnectionCredentials{
		DbServer:   "mongodb://localhost:27017",
		DbName:     "gmongo",
		DbPassword: "",
	}

	// act
	client, err := ConnectUsingCredentials(credentials)
	if err != nil {
		t.Error(err)
	}

	// check if connected
	if !client.IsConnected() {
		t.Error("Not connected")
	}

	// close connection
	err = client.MongoClient.Disconnect(context.TODO())
	if err != nil {
		t.Error(err)
	}
}
