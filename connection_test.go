package gmongo

import (
	"context"
	"os"
	"testing"
)

// testConnectToDb opens a test connection. Override the URI by setting
// GMONGO_TEST_URI (e.g. point at a replica set on a different port).
func testConnectToDb() *Client {
	uri := os.Getenv("GMONGO_TEST_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	client, err := ConnectUsingString(uri, "gmongo")
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
