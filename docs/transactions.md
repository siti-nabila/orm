# Transactions And Locking

## Transaction Ownership

The caller owns transaction boundaries. ORM write methods do not auto-commit or
auto-rollback. You decide when to call `Commit` or `Rollback`.

```go
sqlTx, err := conn.BeginTx(ctx, nil)
if err != nil {
    return err
}

tx := orm.NewSqlTransactionAdapter(ctx, sqlTx, dialect.NewPostgres(), config.Config{})

if err := tx.Create(&user); err != nil {
    tx.Rollback()
    return err
}

return tx.Commit()
```

`SqlTransactionAdapter` exposes:

- `Create`
- `Update`
- `CreateBulk`
- `UpdateBulk`
- `DryRunUpdateBulk`
- `CreateWith`
- `UseModel`
- `TryLock`
- `Commit`
- `Rollback`
- `SetLogger`

`Commit` and `Rollback` release acquired locks before ending the transaction.

`UseModel` keeps the transaction context and executor while exposing the query
builder. It can be used for chained model updates:

```go
result, err := tx.
    UseModel(&ProfileUpdate{
        TenantID:    7,
        UserID:      42,
        DisplayName: "Nabila",
    }).
    Updates()
if err != nil {
    tx.Rollback()
    return err
}

if _, err := result.RowsAffected(); err != nil {
    tx.Rollback()
    return err
}

return tx.Commit()
```

`Updates` does not commit or roll back automatically. Transaction ownership
remains with the caller.

## Advisory Locks

`TryLock(ctx, key)` is non-blocking. It normalizes the key, checks whether the
same adapter already acquired it, then delegates to the dialect.

```go
locked, err := tx.TryLock(ctx, "user:123")
if err != nil {
    tx.Rollback()
    return err
}
if !locked {
    tx.Rollback()
    return fmt.Errorf("lock not acquired")
}
```

### PostgreSQL

PostgreSQL uses `pg_try_advisory_xact_lock($1)` with a hash of the lock key.
The lock is transaction-scoped, so no explicit release query is needed.

### MySQL

MySQL uses `GET_LOCK(?, 0)` and releases acquired locks with
`RELEASE_LOCK(?)` during `Commit` or `Rollback`.

### Oracle

Oracle uses `DBMS_LOCK.ALLOCATE_UNIQUE` and `DBMS_LOCK.REQUEST` with
`release_on_commit => TRUE`. The implementation treats return code `0` as
acquired and `1` as not acquired.

## Lock Logging

Set `config.Config{LogLockQuery: true}` to log lock queries through the
configured logger.
