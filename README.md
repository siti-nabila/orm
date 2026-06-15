# 🚀 Go Native ORM (Lightweight & Dialect-Aware)

A lightweight, performant, and extensible ORM built in Go, designed to support multiple SQL dialects (PostgreSQL, MySQL, Oracle) with minimal reflection overhead via metadata caching.

---

## ✨ Features

- 🔌 Multi-dialect support (PostgreSQL, MySQL, Oracle)
- ⚡ Metadata caching (minimize reflection cost)
- 🧱 Query Builder (SELECT with chaining)
- 📝 Create & Update API (struct & map based)
- 🔁 Context propagation (per-request safe)
- 🪵 Advanced logging system:
  - execution log
  - dry run log
  - lock log (optional)
- 🔒 Transaction-level locking (non-blocking, dialect-aware)
- 🧪 Dry Run mode (perfect for sqlmock & debugging)
- 🧩 Adapter-based architecture

---

## 📦 Installation

```bash
go get github.com/siti-nabila/orm
```

---

## 🔍 READ (SELECT)

```go
db.UseModel(User{}).
   Where("email = ?", email).
   Limit(1).
   Scan(&result)
```

### Offset Pagination

Database pagination applies offset and limit in SQL. It fetches one extra row
to determine `HasNext` without running a count query:

```go
var users []User

meta, err := db.UseModel(User{}).
    Where("active = ?", true).
    OrderBy("id ASC").
    Paginate(&users, pagination.Params{
        Page:  2,
        Limit: 20,
    })
```

Use `WithTotal()` to calculate an exact total in the same query with
`COUNT(*) OVER()`. `TotalItems` is `nil` when the requested page has no rows.
`WithTotal()` does not support joins because joined rows may produce a
misleading total.

```go
meta, err := db.UseModel(User{}).
    Where("active = ?", true).
    OrderBy("id ASC").
    WithTotal().
    Paginate(&users, pagination.Params{Page: 1, Limit: 20})
```

The default page limit is `10` and the maximum is `1000`:

```go
pagination.Params{Page: 1}           // effective limit: 10
pagination.Params{Page: 1, Limit: -1} // effective limit: 1000
pagination.Params{Page: 1, Limit: 5000} // clamped to 1000
```

Both limits can be configured per ORM instance:

```go
cfg := config.Config{
    Pagination: pagination.Config{
        DefaultLimit: 25,
        MaxLimit:     500,
    },
}
```

### In-Memory Pagination

Use the same pagination parameters and metadata for slices already loaded in
memory:

```go
result, err := pagination.FromSlice(users).
    Filter(func(user User) bool {
        return user.Active
    }).
    Sort(func(a, b User) bool {
        return a.ID < b.ID
    }).
    Paginate(pagination.Params{Page: 1, Limit: 20})
```

Use `FromSliceWithConfig()` to customize its default and maximum limit:

```go
result, err := pagination.FromSliceWithConfig(
    users,
    pagination.Config{DefaultLimit: 25, MaxLimit: 500},
).Paginate(pagination.Params{Page: 1})
```

In-memory processing follows `Filter → stable Sort → offset → limit` and does
not modify the input slice.

---

## 📝 CREATE

```go
tx := orm.NewSqlTransactionAdapter(ctx, sqlTx, dialect, cfg)

if err := tx.Create(&user); err != nil {
    tx.Rollback()
    return err
}

return tx.Commit()
```

---

## ✏️ UPDATE

```go
err := tx.Update(&user)
```

or

```go
err := tx.Update(&user, map[string]any{
    "email": "new@email.com",
})
```

---

## 🔒 TRANSACTION LOCKING (Non-Blocking)

```go
locked, err := tx.TryLock(ctx, "user:123")
if err != nil {
    tx.Rollback()
    return err
}

if !locked {
    tx.Rollback()
    return errors.New("resource is locked")
}

err = tx.Update(&user)

return tx.Commit()
```

---

## 🧪 DRY RUN (Testing & Debugging)

### Create

```go
res, err := orm.RunDryCreate(ctx, &user)
```

### Update

```go
res, err := orm.RunDryUpdate(ctx, &user)
```

### Select

```go
res, err := db.UseModel(User{}).
    Where("email = ?", email).
    DryRun()
```

---

## 🪵 LOGGING CONFIG

```go
cfg := config.Config{
    EnableDebug:    true,
    LogDryRunQuery: true,
    LogLockQuery:   true,
}
```

---

## 📌 Summary

- Lightweight ORM
- Multi-dialect support
- Transaction locking
- DryRun support for testing
- Clean architecture
