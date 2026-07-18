# Contoh

## Membaca User

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

## Mencari Satu User

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

## Method Repositori dengan Paginasi

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

## Respons API dengan Paginasi

Request frontend:

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

Respons backend:

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

`items` berisi baris untuk halaman saat ini. `PageData[T]` juga menyertakan jumlah
total baris, halaman, limit, total halaman, serta flag next/prev.

Hasil kosong:

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

Halaman terakhir:

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

Saat menerima filter frontend dinamis, petakan field request ke kolom database
yang dikenal dan jangan pernah menerima SQL mentah dari client. Untuk rentang
timestamp, parse string tanggal sebelum memanggil repositori dan gunakan batas
akhir eksklusif.

## Menulis dalam Transaksi

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

## Paginasi Slice dalam Memori

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

## Dry Run Query dengan Paginasi

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
