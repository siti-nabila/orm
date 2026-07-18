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
update := &ProfileUpdate{
    TenantID:    7,
    UserID:      42,
    DisplayName: "Nabila",
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
| PostgreSQL | `SET display_name = $1 WHERE tenant_id = $2 AND user_id = $3` |
| MySQL | `SET display_name = ? WHERE tenant_id = ? AND user_id = ?` |
| Oracle | `SET display_name = :display_name WHERE tenant_id = :tenant_id AND user_id = :user_id` |

For all three dialects:

```go
result.Args == []any{"Nabila", uint64(7), uint64(42)}
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

Long values in the rendered default log are truncated, and PostgreSQL array
args are rendered as `ARRAY[...]`. `DryRunResult.Query`, `DryRunResult.Args`, and
`DryRunResult.Mode` are not modified by log formatting. In particular, the args
returned by DryRun remain the original parameter values supplied by the caller.
