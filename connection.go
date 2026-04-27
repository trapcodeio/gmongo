package gmongo

import (
	"context"
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

// Tx - A transaction handle passed to the callback of Client.Transaction.
// Use Tx.Context() to enroll raw mongo-driver calls in the transaction, or
// Tx.Collection(name) for native access to a collection that has no Model.
type Tx struct {
	sc mongo.SessionContext
	db *mongo.Database
}

// Context returns the session context. Pass it to any raw mongo-driver call
// to enroll that op in the transaction.
func (t *Tx) Context() mongo.SessionContext { return t.sc }

// Database returns the *mongo.Database the transaction is running on.
func (t *Tx) Database() *mongo.Database { return t.db }

// Collection is sugar for tx.Database().Collection(name) — convenient for
// native ops on collections that don't have a gmongo Model.
func (t *Tx) Collection(name string) *mongo.Collection { return t.db.Collection(name) }

// Transaction runs fn inside a MongoDB transaction. The session lifecycle is
// managed for the caller. Return an error from fn to abort; nil to commit.
//
// Inside fn, use Model[T].WithTx(tx) to get a transaction-bound model, or
// tx.Context() / tx.Collection(name) for native mongo-driver access.
//
// Note: MongoDB transactions require a replica set or sharded cluster.
func (c *Client) Transaction(fn func(tx *Tx) error, opts ...*options.TransactionOptions) error {
	session, err := c.MongoClient.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(context.TODO())

	_, err = session.WithTransaction(
		context.TODO(),
		func(sc mongo.SessionContext) (interface{}, error) {
			return nil, fn(&Tx{sc: sc, db: c.Database})
		},
		opts...,
	)
	return err
}
