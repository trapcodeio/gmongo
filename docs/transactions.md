# Transactions in gmongo

This document explains MongoDB transactions from the ground up: what they are, how the official Go driver exposes them, and how gmongo wraps that in a way that keeps the rest of the gmongo API ergonomic.

If you've never used MongoDB transactions before, start at the top. If you just want the gmongo API, jump to [The gmongo way](#the-gmongo-way).

---

## What is a transaction?

A transaction is a **group of database operations that either all succeed or all fail**. If anything goes wrong partway through, every change made so far is rolled back as if nothing happened. The classic example is a money transfer:

1. Read account A's balance
2. Subtract `$100` from A
3. Add `$100` to B
4. Write a transfer log row

If step 3 fails after step 2 succeeds, you've just destroyed money. A transaction guarantees that doesn't happen — either all four lines land in the database, or none of them do.

In MongoDB, single-document writes are already atomic. Transactions exist for the case where you need atomicity *across multiple documents or multiple collections*.

---

## Prerequisite: a replica set

MongoDB transactions only work against a **replica set or a sharded cluster**. They do **not** work against a standalone `mongod`.

The reason is that transactions are implemented on top of the oplog (the operation log used for replication). A standalone server has no oplog, so it has nothing to commit or abort against.

This isn't usually a problem in production — nobody runs production MongoDB as a standalone. But for local development, you need to start your `mongod` with `--replSet <name>` and run `rs.initiate()` once. A single-node replica set is enough; you don't need three servers.

If gmongo's transaction tests run against a standalone, they will **skip** with a clear message rather than fail.

---

## How the native mongo-driver handles a transaction

Here is the same money-transfer flow written using `go.mongodb.org/mongo-driver` directly, with no helpers.

```go
// 1. Start a session.
session, err := client.StartSession()
if err != nil {
    return err
}
defer session.EndSession(context.TODO())

// 2. Run the work inside session.WithTransaction.
//    The driver handles BeginTransaction / CommitTransaction / AbortTransaction,
//    and will retry the callback on transient errors.
_, err = session.WithTransaction(context.TODO(),
    func(sc mongo.SessionContext) (interface{}, error) {

        // 3. EVERY db operation inside the transaction must receive sc
        //    (the SessionContext) instead of a plain context. This is what
        //    enrols the operation in the transaction.
        var from Account
        err := db.Collection("accounts").
            FindOne(sc, bson.M{"_id": fromID}).Decode(&from)
        if err != nil {
            return nil, err
        }
        if from.Balance < amount {
            return nil, errors.New("insufficient funds")
        }

        _, err = db.Collection("accounts").UpdateOne(sc,
            bson.M{"_id": fromID},
            bson.M{"$inc": bson.M{"balance": -amount}})
        if err != nil {
            return nil, err
        }

        _, err = db.Collection("accounts").UpdateOne(sc,
            bson.M{"_id": toID},
            bson.M{"$inc": bson.M{"balance": amount}})
        if err != nil {
            return nil, err
        }

        _, err = db.Collection("transfers").InsertOne(sc,
            bson.M{"from": fromID, "to": toID, "amount": amount})
        return nil, err
    })

return err
```

### What's going on under the hood

- **Session** — A session is the thing that owns the transaction. You start one, run a transaction (or several) inside it, and end it.
- **`session.WithTransaction`** — Wraps `BeginTransaction` + `CommitTransaction` + `AbortTransaction`. If your callback returns an error, the transaction is aborted. If it returns nil, the transaction is committed.
- **Automatic retries** — `WithTransaction` retries your callback automatically when MongoDB returns a `TransientTransactionError` or `UnknownTransactionCommitResult`. **This means your callback can run more than once**, so it must be idempotent (no in-memory mutation that you don't want repeated).
- **`mongo.SessionContext`** — A regular `context.Context` with a session attached. Passing it to `Collection.FindOne` / `UpdateOne` / etc. is what tells MongoDB "this operation is part of the transaction." If you pass a plain `context.Background()` or `context.TODO()` instead, the operation runs *outside* the transaction and won't be rolled back.

### What hurts

- **Boilerplate** — `StartSession`, `defer EndSession`, the `WithTransaction` wrapper, the awkward `(interface{}, error)` return signature.
- **`sc` everywhere** — Every single database call needs the session context threaded through. Forgetting it on one line silently breaks the transaction (that operation runs outside the tx and won't be rolled back).
- **Type safety dies** — gmongo gives you typed models like `Account`. Native code drops back to `db.Collection("accounts").FindOne(sc, filter).Decode(&x)`. You lose `Find[T]`, `FindOneById`, `PublicFields`, `Helpers`, projection builders — everything gmongo gave you.

---

## The gmongo way

gmongo adds three small pieces and otherwise stays out of your way:

### 1. `Client.Transaction(fn)`

Hides the session lifecycle. Return an error from `fn` to abort, nil to commit.

```go
err := client.Transaction(func(tx *gmongo.Tx) error {
    // ... your work ...
    return nil
})
```

Internally this is just `StartSession` → `WithTransaction` → `EndSession`, exactly like the native version. You still get automatic retries on transient errors.

### 2. `Model[T].WithTx(tx)`

Returns a **clone** of the model bound to the transaction. The original model is unchanged. Any operation on the cloned model is enrolled in the transaction automatically — no more threading `sc` through every call.

```go
accounts := AccountModel.WithTx(tx)   // tx-bound clone
accounts.UpdateOne(...)                // enrolled in tx
AccountModel.UpdateOne(...)            // unchanged: NOT in tx
```

This mirrors gmongo's existing `Model.Helpers(model)` pattern: a method that returns a wrapper around the same underlying state. Nothing magical — just a copy of the struct with an extra field set.

### 3. Native escape hatch on `Tx`

Sometimes you have a collection that doesn't have a gmongo `Model` (maybe legacy, maybe one-off). The `Tx` handle exposes both the session context and the database directly:

```go
tx.Context()              // mongo.SessionContext — pass to any raw mongo-driver op
tx.Database()             // *mongo.Database — the live DB the tx is running on
tx.Collection("things")   // shortcut for tx.Database().Collection("things")
```

You can mix and match freely:

```go
err := client.Transaction(func(tx *gmongo.Tx) error {
    // gmongo modeled access:
    user, err := UserModel.WithTx(tx).FindOneById(id)
    if err != nil { return err }

    // raw, non-modeled access in the same transaction:
    _, err = tx.Collection("audit_log").InsertOne(tx.Context(), bson.M{
        "userID": id, "event": "login",
    })
    return err
})
```

The audit-log insert is enrolled in the same transaction — both rolled back together if anything fails.

---

## The money-transfer example, in gmongo

Same flow as the native version above. Compare line by line.

```go
err := client.Transaction(func(tx *gmongo.Tx) error {
    accounts  := AccountModel.WithTx(tx)
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
        bson.M{"$inc": bson.M{"balance": -amount}}); err != nil {
        return err
    }
    if _, err := accounts.UpdateOne(
        bson.M{"_id": toID},
        bson.M{"$inc": bson.M{"balance": amount}}); err != nil {
        return err
    }

    _, err = transfers.InsertOne(&Transfer{
        ID: gmongo.NewId(), From: fromID, To: toID, Amount: amount,
    })
    return err
})
```

What changed:

- No `StartSession`, no `defer EndSession`, no `WithTransaction(...)` wrapper.
- No `sc` threaded through every call.
- `from` is a typed `*Account`, not a `bson.M` you have to `Decode` yourself.
- `accounts.UpdateOne` is the same method you call outside transactions — not a different "tx variant."
- The whole closure returns a plain `error`, not `(interface{}, error)`.

---

## API reference

```go
// in package gmongo:

type Tx struct { /* ... */ }

func (t *Tx) Context() mongo.SessionContext
func (t *Tx) Database() *mongo.Database
func (t *Tx) Collection(name string) *mongo.Collection

func (c *Client) Transaction(
    fn func(tx *Tx) error,
    opts ...*options.TransactionOptions,
) error

func (coll *Model[T]) WithTx(tx *Tx) *Model[T]
```

`options.TransactionOptions` is the standard mongo-driver type and lets you set things like read concern, write concern, and read preference for the transaction.

---

## Things to know

### Your callback may run more than once

`session.WithTransaction` (which `Client.Transaction` uses internally) automatically retries on transient errors and unknown commit results. **Your callback must be idempotent**. Specifically:

- Don't mutate in-memory state (counters, slices, etc.) inside the callback in ways you don't want repeated.
- Don't perform side effects that aren't transactional (sending an email, calling an external API, writing to a file). Move those *after* the `Transaction(...)` call returns nil.
- Reads inside the transaction will see the same snapshot each retry — that's fine.

### Helpers inherit the transaction binding

`Model.Helpers(doc)` returns a `ModelHelper` whose `Update`, `UpdateRaw`, and `Delete` go through the underlying model's methods. So this works as expected:

```go
client.Transaction(func(tx *gmongo.Tx) error {
    accounts := AccountModel.WithTx(tx)
    acc, err := accounts.FindOneById(id)
    if err != nil { return err }

    helper := accounts.Helpers(acc)   // helper is implicitly tx-bound
    _, err = helper.Update(bson.M{"verified": true})
    return err
})
```

### `WithTx` doesn't mutate the original model

`AccountModel.WithTx(tx)` returns a *new* `*Model[T]` with the session context set. The original `AccountModel` is unchanged and continues to behave like a non-transactional model. You can safely use both inside the same closure if you really need to (though it's rarely a good idea).

### You can't nest transactions

MongoDB does not support nested transactions. Calling `Client.Transaction(...)` inside another `Client.Transaction(...)` will start a separate session, which is almost never what you want. Don't do it.

### Transactions have a time limit

By default, MongoDB aborts a transaction that runs for more than 60 seconds. Keep transactions short. If you have long-running work, do the long part outside the transaction and only the database writes inside it.

### Transactions need a replica set

Repeating from the top, because it bites everyone once: standalone `mongod` doesn't support transactions. Use `mongod --replSet rs0` + `rs.initiate()` for local development. A single-node replica set works fine.
