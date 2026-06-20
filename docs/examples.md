# Examples

## Read Users

```go
func FindActiveUsers(ctx context.Context, db *orm.SqlQueryAdapter) ([]User, error) {
    var users []User

    builder, err := db.UseModel(User{}).
        WhereOp("status", orm.OpEqual, "ACTIVE")
    if err != nil {
        return nil, err
    }

    err = builder.
        OrderBy("created_at DESC").
        Scan(&users)
    if err != nil {
        return nil, err
    }

    return users, nil
}
```

## Find One User

```go
func FindUserByEmail(ctx context.Context, db *orm.SqlQueryAdapter, email string) (*User, error) {
    var user User

    err := db.UseModel(User{}).
        Where("email = ?", email).
        First(&user)
    if err != nil {
        return nil, err
    }

    return &user, nil
}
```

## Paginated Repository Method

```go
type UserRepository struct {
    db *orm.SqlQueryAdapter
}

func (r *UserRepository) FindPaginated(ctx context.Context, page, perPage int) ([]User, *orm.PageMeta, error) {
    var users []User

    builder, err := r.db.
        UseModel(User{}).
        WhereOp("status", orm.OpEqual, "ACTIVE")
    if err != nil {
        return nil, nil, err
    }

    pageMeta, err := builder.
        OrderBy("created_at DESC").
        ScanPaginate(ctx, &users, orm.PaginationOptions{
            Page:    page,
            PerPage: perPage,
        })
    if err != nil {
        return nil, nil, err
    }

    return users, pageMeta, nil
}
```

## Paginated API Response

Frontend request:

```json
{
  "page": 1,
  "perPage": 20,
  "joinDateFrom": "2026-01-01",
  "joinDateTo": "2026-03-22",
  "sortField": "joinDate",
  "sortDirection": "DESC"
}
```

Backend response:

```json
{
  "items": [
    {
      "id": 1,
      "name": "Nabila",
      "email": "nabila@example.com",
      "status": "ACTIVE",
      "joinDate": "2026-01-10"
    }
  ],
  "total": 55,
  "page": 1,
  "limit": 20,
  "total_pages": 3,
  "has_next": true,
  "has_prev": false
}
```

`items` contains the rows for the current page. `PageData[T]` also includes
the total row count, page, limit, total pages, and next/prev flags.

Empty result:

```json
{
  "items": [],
  "total": 0,
  "page": 1,
  "limit": 20,
  "total_pages": 0,
  "has_next": false,
  "has_prev": false
}
```

Last page:

```json
{
  "items": [
    {
      "id": 41,
      "name": "Last User",
      "email": "last.user@example.com",
      "status": "ACTIVE",
      "joinDate": "2026-03-22"
    }
  ],
  "total": 41,
  "page": 3,
  "limit": 20,
  "total_pages": 3,
  "has_next": false,
  "has_prev": true
}
```

When accepting dynamic frontend filters, map request fields to known database
columns and never accept raw SQL from the client. For timestamp ranges, parse
date strings before the repository call and use an exclusive end boundary.

## Write In A Transaction

```go
func CreateUser(ctx context.Context, conn *sql.DB, user *User) error {
    sqlTx, err := conn.BeginTx(ctx, nil)
    if err != nil {
        return err
    }

    tx := orm.NewSqlTransactionAdapter(ctx, sqlTx, dialect.NewPostgres(), config.Config{})

    if err := tx.Create(user); err != nil {
        tx.Rollback()
        return err
    }

    return tx.Commit()
}
```

## In-Memory Slice Pagination

```go
func ActiveUserPage(users []User) (pagination.PageData[User], error) {
    return pagination.
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
}
```

## Dry Run A Paginated Query

```go
func PrintPaginatedSQL(db *orm.SqlQueryAdapter) error {
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

    fmt.Println(result.Count.Query, result.Count.Args)
    fmt.Println(result.Data.Query, result.Data.Args)
    return nil
}
```
