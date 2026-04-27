package gmongo

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/**
================== Define Models ==================
*/

type Account struct {
	ID      primitive.ObjectID `bson:"_id"`
	Owner   string             `bson:"owner"`
	Balance int                `bson:"balance"`
}

func (a *Account) GetID() primitive.ObjectID { return a.ID }

type Transfer struct {
	ID     primitive.ObjectID `bson:"_id"`
	From   primitive.ObjectID `bson:"from"`
	To     primitive.ObjectID `bson:"to"`
	Amount int                `bson:"amount"`
}

func (t *Transfer) GetID() primitive.ObjectID { return t.ID }

// requireReplicaSet skips the test if the connected mongo isn't a replica set
// or sharded cluster — transactions require one of those.
func requireReplicaSet(t *testing.T, client *Client) {
	t.Helper()
	var hello bson.M
	err := client.Database.RunCommand(context.TODO(), bson.M{"hello": 1}).Decode(&hello)
	if err != nil {
		t.Skipf("could not run hello command: %v", err)
	}
	if _, isReplSet := hello["setName"]; isReplSet {
		return
	}
	if msg, isMongos := hello["msg"]; isMongos && msg == "isdbgrid" {
		return
	}
	t.Skip("transactions require a replica set or sharded cluster; skipping")
}

func TestTransaction(t *testing.T) {
	client := testConnectToDb()
	requireReplicaSet(t, client)

	AccountModel := MakeModel[*Account](client.Database, "tx_accounts")
	TransferModel := MakeModel[*Transfer](client.Database, "tx_transfers")

	reset := func() (a, b Account) {
		_, _ = AccountModel.Native().DeleteMany(context.TODO(), bson.M{})
		_, _ = TransferModel.Native().DeleteMany(context.TODO(), bson.M{})
		_, _ = client.Database.Collection("tx_audit").DeleteMany(context.TODO(), bson.M{})

		a = Account{ID: NewId(), Owner: "alice", Balance: 1000}
		b = Account{ID: NewId(), Owner: "bob", Balance: 500}
		if _, err := AccountModel.InsertMany([]*Account{&a, &b}); err != nil {
			t.Fatal(err)
		}
		return
	}

	transfer := func(tx *Tx, fromID, toID primitive.ObjectID, amount int) error {
		accounts := AccountModel.WithTx(tx)
		transfers := TransferModel.WithTx(tx)

		from, err := accounts.FindOneById(fromID)
		if err != nil {
			return err
		}
		if from.Balance < amount {
			return errors.New("insufficient funds")
		}

		if _, err := accounts.UpdateOne(
			bson.M{"_id": fromID},
			bson.M{"$inc": bson.M{"balance": -amount}},
		); err != nil {
			return err
		}
		if _, err := accounts.UpdateOne(
			bson.M{"_id": toID},
			bson.M{"$inc": bson.M{"balance": amount}},
		); err != nil {
			return err
		}

		_, err = transfers.InsertOne(&Transfer{
			ID: NewId(), From: fromID, To: toID, Amount: amount,
		})
		return err
	}

	t.Run("Commits on nil error", func(t *testing.T) {
		alice, bob := reset()

		err := client.Transaction(func(tx *Tx) error {
			return transfer(tx, alice.ID, bob.ID, 200)
		})
		assert.NoError(t, err)

		aliceAfter, _ := AccountModel.FindOneById(alice.ID)
		bobAfter, _ := AccountModel.FindOneById(bob.ID)
		assert.Equal(t, 800, aliceAfter.Balance)
		assert.Equal(t, 700, bobAfter.Balance)

		count, _ := TransferModel.Count(bson.M{})
		assert.Equal(t, int64(1), count)
	})

	t.Run("Rolls back on error", func(t *testing.T) {
		alice, bob := reset()

		err := client.Transaction(func(tx *Tx) error {
			// move money first, then fail — the prior writes must roll back
			if err := transfer(tx, alice.ID, bob.ID, 200); err != nil {
				return err
			}
			return errors.New("boom")
		})
		assert.EqualError(t, err, "boom")

		aliceAfter, _ := AccountModel.FindOneById(alice.ID)
		bobAfter, _ := AccountModel.FindOneById(bob.ID)
		assert.Equal(t, 1000, aliceAfter.Balance)
		assert.Equal(t, 500, bobAfter.Balance)

		count, _ := TransferModel.Count(bson.M{})
		assert.Equal(t, int64(0), count)
	})

	t.Run("Insufficient funds aborts", func(t *testing.T) {
		alice, bob := reset()

		err := client.Transaction(func(tx *Tx) error {
			return transfer(tx, alice.ID, bob.ID, 9999)
		})
		assert.EqualError(t, err, "insufficient funds")

		aliceAfter, _ := AccountModel.FindOneById(alice.ID)
		assert.Equal(t, 1000, aliceAfter.Balance)
	})

	t.Run("Native escape via tx.Collection rolls back with the tx", func(t *testing.T) {
		alice, bob := reset()

		err := client.Transaction(func(tx *Tx) error {
			if err := transfer(tx, alice.ID, bob.ID, 100); err != nil {
				return err
			}
			// Native, non-modeled collection — must enroll via tx.Context()
			_, err := tx.Collection("tx_audit").InsertOne(tx.Context(), bson.M{
				"event": "transfer", "amount": 100,
			})
			if err != nil {
				return err
			}
			return errors.New("rollback please")
		})
		assert.Error(t, err)

		auditCount, _ := client.Database.Collection("tx_audit").
			CountDocuments(context.TODO(), bson.M{})
		assert.Equal(t, int64(0), auditCount, "native insert should have rolled back")
	})

	t.Run("Native escape via tx.Database matches", func(t *testing.T) {
		alice, bob := reset()

		err := client.Transaction(func(tx *Tx) error {
			_, err := tx.Database().Collection("tx_audit").InsertOne(tx.Context(), bson.M{
				"event": "noop",
			})
			if err != nil {
				return err
			}
			return transfer(tx, alice.ID, bob.ID, 50)
		})
		assert.NoError(t, err)

		auditCount, _ := client.Database.Collection("tx_audit").
			CountDocuments(context.TODO(), bson.M{})
		assert.Equal(t, int64(1), auditCount)
	})

	t.Run("ModelHelper inherits transaction binding", func(t *testing.T) {
		alice, _ := reset()

		err := client.Transaction(func(tx *Tx) error {
			a, err := AccountModel.WithTx(tx).FindOneById(alice.ID)
			if err != nil {
				return err
			}
			helper := AccountModel.WithTx(tx).Helpers(a)
			if _, err := helper.Update(bson.M{"balance": 7}); err != nil {
				return err
			}
			return errors.New("rollback")
		})
		assert.Error(t, err)

		// helper.Update must have rolled back along with the tx
		aliceAfter, _ := AccountModel.FindOneById(alice.ID)
		assert.Equal(t, 1000, aliceAfter.Balance)
	})

	t.Run("Original model is unchanged after WithTx", func(t *testing.T) {
		// WithTx must return a clone — using AccountModel directly outside
		// the closure must not be tx-bound.
		_, _ = reset()

		_ = client.Transaction(func(tx *Tx) error {
			_ = AccountModel.WithTx(tx)
			return nil
		})

		// AccountModel.txCtx must still be nil — verify by doing a normal op:
		count, err := AccountModel.Count(bson.M{})
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})
}
