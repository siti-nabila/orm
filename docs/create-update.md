# Create, Update, And Bulk Writes

Write APIs are exposed on `*orm.ORM` and `*orm.SqlTransactionAdapter`. The
transaction adapter is the most direct user-facing entrypoint when you already
own a `*sql.Tx`.

## Create

```go
tx := orm.NewSqlTransactionAdapter(ctx, sqlTx, dialect.NewPostgres(), config.Config{})

user := User{
    Name:   "Nabila",
    Email:  "nabila@example.com",
    Status: "ACTIVE",
}

if err := tx.Create(&user); err != nil {
    tx.Rollback()
    return err
}

return tx.Commit()
```

`Create` expects a pointer to a struct. If the dialect supports returning and a
primary key is mapped, the generated primary key can be scanned back into the
model.

## Update

Update accepts a pointer to a struct. Without an explicit field map, it builds
the update from mapped struct values and the primary key.

```go
user.Email = "new@example.com"

if err := tx.Update(&user); err != nil {
    tx.Rollback()
    return err
}
```

You can pass a map to update selected columns:

```go
err := tx.Update(&user, map[string]any{
    "email":  "new@example.com",
    "status": "ACTIVE",
})
```

The map keys are column names.

## Advanced Insert: CreateWith

`CreateWith` returns a `CreateCommand` for returning columns and conflict
handling.

```go
err := tx.CreateWith(&user).
    WithReturning("id", "email").
    Scan(&user)
```

Use `ScanInto` when you want explicit destination variables:

```go
var id int64
var email string

err := tx.CreateWith(&user).
    WithReturning("id", "email").
    ScanInto(&id, &email)
```

`Exec` is for create commands without returning columns:

```go
err := tx.CreateWith(&user).Exec()
```

If returning columns are configured, use `Scan` or `ScanInto`; `Exec` rejects
returning mode.

## Conflict Handling

`WithOnConflict` accepts `orm.OnConflict`.

```go
err := tx.CreateWith(&user).
    WithOnConflict(orm.OnConflict{
        TargetColumns: []string{"email"},
        DoUpdates:     []string{"name", "status"},
    }).
    Exec()
```

Custom assignments are available with `orm.Value` and `orm.Inc`:

```go
err := tx.CreateWith(&user).
    WithOnConflict(orm.OnConflict{
        TargetColumns: []string{"email"},
        Assignments: []orm.ConflictAssignment{
            {Column: "status", Expr: orm.Value("ACTIVE")},
            {Column: "login_count", Expr: orm.Inc("login_count", 1)},
        },
    }).
    Exec()
```

`DoNothing` is also supported:

```go
err := tx.CreateWith(&user).
    WithOnConflict(orm.OnConflict{
        TargetColumns: []string{"email"},
        DoNothing:     true,
    }).
    Exec()
```

Do not combine `DoNothing` with update assignments.

## Bulk Insert

```go
users := []User{
    {Name: "A", Email: "a@example.com"},
    {Name: "B", Email: "b@example.com"},
}

err := tx.CreateBulk(users)
```

Bulk insert accepts a slice of structs or pointers to structs. Rows must resolve
to a compatible table and layout. PostgreSQL uses returning behavior for primary
keys; MySQL and Oracle execute the bulk statement.

## Bulk Update

```go
err := tx.UpdateBulk(users)
```

Bulk update requires primary keys so each row can be matched. The transaction
adapter also exposes `DryRunUpdateBulk`.

## Not Implemented

Delete APIs are not part of this repository.
