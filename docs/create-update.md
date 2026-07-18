# Create, Update, And Bulk Writes

Write APIs are exposed on `*orm.ORM` and `*orm.SqlTransactionAdapter`. The
transaction adapter is the most direct user-facing entrypoint when you already
own a `*sql.Tx`.

## Create

```go
tx := orm.NewSqlTransactionAdapter(ctx, sqlTx, dialect.NewPostgres(), config.Config{})

user := User{
    Name:   "joko",
    Email:  "joko@example.com",
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

### Update Models Without A Primary Key

An update-specific model can mark one or more columns as update conditions with
the `where` tag. This is useful for tables or write models that do not expose a
primary key.

```go
type ProfileUpdate struct {
    TenantID    uint64 `sql:"column:tenant_id;where"`
    UserID      uint64 `sql:"column:user_id;where"`
    DisplayName string `sql:"column:display_name"`
}

func (ProfileUpdate) TableName() string {
    return "user_profiles"
}

update := &ProfileUpdate{
    TenantID:    7,
    UserID:      42,
    DisplayName: "Nabila",
}

if err := tx.Update(update); err != nil {
    tx.Rollback()
    return err
}
```

The generated condition combines all tagged columns with `AND`:

```sql
UPDATE user_profiles
SET display_name = $1
WHERE tenant_id = $2 AND user_id = $3
```

The args remain parameterized and ordered as:

```go
[]any{"Nabila", uint64(7), uint64(42)}
```

If a model contains a `primaryKey`, the primary key remains the condition source
for `Update`; tagged `where` columns are only the fallback when no primary key is
defined. A nil pointer used by a tagged condition is rejected. Scalar zero
values such as `0`, `false`, and `""` remain valid condition values, so callers
must set them intentionally.

When passing an update map, a tagged `where` column cannot also be updated:

```go
// Rejected because user_id is part of the update condition.
err := tx.Update(update, map[string]any{
    "user_id": uint64(84),
})
```

### Chained Model Updates

`UseModel` is available on the transaction adapter for update chaining. The
model must be a non-nil pointer to a struct.

```go
result, err := tx.
    UseModel(update).
    Where("tenant_id = ?", tenantID).
    Updates()
if err != nil {
    tx.Rollback()
    return err
}

rows, err := result.RowsAffected()
```

`Updates()` includes every field with an explicit `sql` tag in `SET`, including
zero values, except fields tagged `primaryKey` or `where`. Prefer a small,
update-specific struct so unrelated zero-value fields are not written.

The condition rules are:

1. An explicit chained condition (`Where`, `OrWhere`, `WhereIn`, grouped
   conditions, and their variants) is the complete `WHERE` source.
2. Without an explicit condition, a non-zero primary key is used.
3. Without a primary key, tagged `where` columns are used.
4. An update with no condition is rejected.

Explicit conditions are not automatically combined with the primary key or
tagged conditions. Include every required tenant or ownership constraint in the
chain.

The behavior is shared by PostgreSQL, MySQL, and Oracle. Existing placeholder
and identifier-quoting configuration remains in effect.

### Update Logging

Update query logging dereferences scalar pointers before interpolation and uses
`database/sql/driver.Valuer` values when available. This only changes rendered
log values; the original query and args sent to the database are unchanged.

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
