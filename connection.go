package gmongo

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strings"
)

type Client struct {
	MongoClient *mongo.Client
	Database    *mongo.Database
	connected   bool
}

type ConnectionCredentials struct {
	DbServer   string
	DbName     string
	DbPassword string
}

func (c *Client) IsConnected() bool {
	return c.connected
}

func ConnectUsingString(connectionString string, database string) (*Client, error) {
	mongoClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(connectionString))
	if err != nil {
		return nil, err
	}

	gmongoClient := &Client{
		MongoClient: mongoClient,
		Database:    mongoClient.Database(database),
		connected:   true,
	}

	return gmongoClient, nil
}

func ConnectUsingCredentials(credentials *ConnectionCredentials) (*Client, error) {
	DbServer := credentials.DbServer
	if DbServer == "" {
		return nil, errors.New("DbServer is empty")
	} else {
		// parse <dbname> and <password> from DbServer
		DbServer = strings.Replace(DbServer, "<dbname>", credentials.DbName, 1)
		DbServer = strings.Replace(DbServer, "<password>", credentials.DbPassword, 1)
	}

	DbName := credentials.DbName
	if DbName == "" {
		return nil, errors.New("DbName is empty")
	}

	return ConnectUsingString(DbServer, DbName)
}
