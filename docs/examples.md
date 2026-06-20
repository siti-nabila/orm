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
func ActiveUserPage(users []User) (pagination.Result[User], error) {
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
