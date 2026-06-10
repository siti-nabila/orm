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
