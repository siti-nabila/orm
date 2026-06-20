# Pagination

The repository has two pagination styles:

- database pagination through query builders
- in-memory slice pagination through `pagination.SlicePaginator`

## ScanPaginate

`ScanPaginate` runs a count query, then a data query with pagination.

```go
var users []User

builder, err := db.
    UseModel(User{}).
    WhereOp("status", orm.OpEqual, "ACTIVE")
if err != nil {
    return err
}

pageInfo, err := builder.
    OrderBy("created_at DESC").
    ScanPaginate(ctx, &users, orm.PaginationOptions{
        Page:    1,
        PerPage: 20,
    })
if err != nil {
    return err
}

fmt.Println(pageInfo.TotalRows)
fmt.Println(pageInfo.TotalPages)
```

`PaginationOptions` is re-exported from package `orm`:

```go
type PaginationOptions struct {
    Page     int
    PerPage  int
    MaxLimit int
}
```

`Page` is normalized to at least `1`. `PerPage` uses the configured default
when zero, rejects negative values below the supported sentinel behavior, and
can be capped with `MaxLimit`.

`PageInfo` is also re-exported from package `orm`:

```go
type PageInfo struct {
    Page       int
    PerPage    int
    TotalRows  int64
    TotalPages int
    HasNext    bool
    HasPrev    bool
}
```

The count query removes ordering and pagination. Simple query shapes use:

```sql
SELECT COUNT(*) FROM users WHERE ...
```

Complex query shapes use a wrapped count:

```sql
SELECT COUNT(*) FROM (SELECT ...) count_table
```

Wrapped count is used for `DISTINCT`, `GROUP BY`, `HAVING`, raw select
expressions, and alias select expressions.

The data query keeps the selected query shape and applies `LIMIT/OFFSET` or the
Oracle pagination clause. `ScanPaginate` does not change existing `Scan`,
`First`, `Limit`, or `Offset` behavior.

## Legacy-Compatible Where

`Where` is still supported. It accepts a SQL fragment and args:

```go
var users []User

pageInfo, err := db.
    UseModel(User{}).
    Where("status = ?", "ACTIVE").
    OrderBy("created_at DESC").
    ScanPaginate(ctx, &users, orm.PaginationOptions{
        Page:    1,
        PerPage: 20,
    })
if err != nil {
    return err
}

fmt.Println(pageInfo.TotalRows)
fmt.Println(pageInfo.TotalPages)
```

## Repository Pattern

```go
type UserRepository struct {
    db *orm.SqlQueryAdapter
}

func (r *UserRepository) FindPaginated(ctx context.Context, page, perPage int) ([]User, *orm.PageInfo, error) {
    var users []User

    builder, err := r.db.
        UseModel(User{}).
        WhereOp("status", orm.OpEqual, "ACTIVE")
    if err != nil {
        return nil, nil, err
    }

    pageInfo, err := builder.
        OrderBy("created_at DESC").
        ScanPaginate(ctx, &users, orm.PaginationOptions{
            Page:    page,
            PerPage: perPage,
        })
    if err != nil {
        return nil, nil, err
    }

    return users, pageInfo, nil
}
```

## Offset Pagination

The older `Paginate` API uses `pagination.Params` and returns
`pagination.Meta`. Without `WithTotal`, it fetches one extra row to determine
`HasNext`. With `WithTotal`, it adds `COUNT(*) OVER() AS __orm_total_items`.

```go
var users []User

meta, err := db.UseModel(User{}).
    Where("status = ?", "ACTIVE").
    OrderBy("id ASC").
    Paginate(&users, pagination.Params{
        Page:  2,
        Limit: 20,
    })
```

`WithTotal()` is available for exact totals in the same query, but joined
queries are rejected for this mode.

## In-Memory Slice Pagination

`pagination.SlicePaginator` paginates data already loaded in memory.

```go
result, err := pagination.
    FromSlice(users).
    Filter(func(user User) bool {
        return user.Status == "ACTIVE"
    }).
    Sort(func(a, b User) bool {
        return a.CreatedAt > b.CreatedAt
    }).
    Paginate(pagination.Params{
        Page:  1,
        Limit: 20,
    })
if err != nil {
    return err
}

fmt.Println(result.Data)
fmt.Println(result.Meta)
```

Public slice pagination types and functions:

- `SlicePaginator`
- `FromSlice`
- `FromSliceWithConfig`
- `Filter`
- `Sort`
- `Paginate`
- `Params`
- `Result`
- `Meta`
- `Config`

`FromSlice` copies the input slice. `Filter` appends non-nil predicates.
`Sort` uses `sort.SliceStable`. `Paginate` uses `NormalizeWithConfig`,
`Offset`, and `BuildMeta`, and copies the returned page data.
