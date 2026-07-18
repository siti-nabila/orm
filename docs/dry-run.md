# Dry Run

Dry run methods return SQL, args, and mode without executing the query.

The result shape is:

```go
type DryRunResult struct {
    Query string
    Args  []any
    Mode  builder.DryRunMode
}
```

Modes are:

- `builder.DryRunModeExec`
- `builder.DryRunModeQuery`
- `builder.DryRunModeQueryRow`

## Create And Update

`*orm.ORM` exposes:

```go
result, err := core.DryRunCreate(&user)
result, err := core.DryRunUpdate(&user)
result, err := core.DryRunUpdate(&user, map[string]any{"email": "new@example.com"})
result, err := core.DryRunUpdateBulk(users)
```

When using `CreateWith`, dry run is on the command:

```go
result, err := tx.CreateWith(&user).
    WithReturning("id").
    DryRun()
```

The transaction adapter currently exposes `DryRunUpdateBulk`.

### Chained Model Update Dry Run

Use `DryRunUpdates` to inspect a chained model update without executing it:

```go
update := &ApprovalLogUpdate{
    CompanyID:   7,
    ReferenceID: "INV-001",
    Flag:        0,
}

result, err := tx.
    UseModel(update).
    DryRunUpdates()
if err != nil {
    return err
}

fmt.Println(result.Query)
fmt.Println(result.Args)
fmt.Println(result.Mode)
```

For the model documented in [Create, Update, and Bulk Writes](create-update.md),
verify the following output shape:

| Dialect | Important query fragment |
| --- | --- |
| PostgreSQL | `SET flag = $1 WHERE company_id = $2 AND reference_id = $3` |
| MySQL | `SET flag = ? WHERE company_id = ? AND reference_id = ?` |
| Oracle | `SET flag = :flag WHERE company_id = :company_id AND reference_id = :reference_id` |

For all three dialects:

```go
result.Args == []any{0, uint64(7), "INV-001"}
result.Mode == builder.DryRunModeExec
```

`DryRunUpdates` uses the same builder and safety checks as `Updates`. It rejects
an update without a condition and does not execute, commit, or roll back a
transaction.

## Query Builder

```go
result, err := db.UseModel(User{}).
    Where("status = ?", "ACTIVE").
    OrderBy("id ASC").
    DryRun()
```

Count and first dry runs:

```go
countResult, err := db.UseModel(User{}).
    Where("status = ?", "ACTIVE").
    DryRunCount()

firstResult, err := db.UseModel(User{}).
    Where("email = ?", email).
    DryRunFirst()
```

## Paginator Dry Run

`DryRunScanPaginate` exposes both queries used by `ScanPaginate`.

```go
builder, err := db.UseModel(User{}).
    WhereOp("status", orm.OpEqual, "ACTIVE")
if err != nil {
    return err
}

result, err := builder.
    OrderBy("created_at DESC").
    DryRunScanPaginate(orm.PaginationOptions{
        Page:    1,
        PerPage: 20,
    })
if err != nil {
    return err
}

fmt.Println(result.Count.Query)
fmt.Println(result.Count.Args)
fmt.Println(result.Count.Mode)

fmt.Println(result.Data.Query)
fmt.Println(result.Data.Args)
fmt.Println(result.Data.Mode)
```

`DryRunPaginate` is the dry run method for the older `Paginate` API based on
`pagination.Params`.

## Dry Run Logging

Set `config.Config{LogDryRunQuery: true}` and configure a logger to emit dry
run logs. The default query adapter installs `logger.DefaultLogger` using
`EnableDebug`; dry-run logging is still controlled by `LogDryRunQuery`.
