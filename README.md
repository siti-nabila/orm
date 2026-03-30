
# 🚀 Go Native ORM (Lightweight & Dialect-Aware)

A lightweight, performant, and extensible ORM built in Go, designed to support multiple SQL dialects (PostgreSQL, MySQL, Oracle) with minimal reflection overhead via metadata caching.

---

## ✨ Features

- Multi-dialect support (PostgreSQL, MySQL, Oracle)
- Metadata caching (minimize reflection cost)
- Query Builder for flexible SELECT queries
- Simple Create & Update API
- Context propagation (per-request safe)
- Built-in SQL logging (debug mode)
- Adapter-based architecture (Query & Transaction separation)

---

## 📦 Installation
```bash
go get github.com/siti-nabila/orm
```
---

## 🏗️ Architecture Overview

### ORM Core
Handles:
- query execution
- logging
- dialect behavior
- metadata parsing (cached)

### SqlQueryAdapter (READ)
Used for SELECT queries:
- inject context
- entry point via UseModel()

### SqlTransactionAdapter (WRITE)
Used for:
- Create
- Update
- Transaction control

### QueryBuilder
Chainable query builder:

```
UseModel(...).Where(...).Limit(...).Scan(...)
```

---

## ⚙️ Model Definition

```
type User struct {
    ID        uint64     `sql:"column:id;primaryKey"`
    Email     string     `sql:"column:email"`
    Password  string     `sql:"column:password"`
    CreatedAt time.Time  `sql:"column:created_at"`
    UpdatedAt time.Time  `sql:"column:updated_at"`
    DeletedAt *time.Time `sql:"column:deleted_at"`
}

func (User) TableName() string {
    return "users"
}
```

---

## 🔍 READ (SELECT)

### Get by Email

```
func (r *authReader) GetByEmail(email string) (result domain.AuthResponse, err error) {
    db := r.Adapter()

    err = db.
        UseModel(store.User{}).
        Where("email = ?", email).
        Limit(1).
        Scan(&result)

    return result, err
}
```

### Multiple Conditions

```
db.UseModel(User{}).
   Where("status = ?", "ACTIVE").
   Where("age > ?", 18).
   OrderBy("created_at DESC").
   Limit(10).
   Scan(&users)
```

---

## 📝 CREATE (INSERT)

```
tx := orm.NewSqlTransactionAdapter(ctx, sqlTx, dialect, cfg)

err := tx.Create(&user)
if err != nil {
    tx.Rollback()
    return err
}

return tx.Commit()
```

Behavior:
- Auto generate INSERT query
- Skip zero primary key
- Support RETURNING / LastInsertId

---

## ✏️ UPDATE

### Struct-based

```
err := tx.Update(&user)
```

### Map-based

```
err := tx.Update(&user, map[string]any{
    "email": "new@email.com",
})
```

---

## 🔄 TRANSACTION FLOW

```
tx := orm.NewSqlTransactionAdapter(ctx, sqlTx, dialect, cfg)
tx.Begin()

if err := tx.Create(&user); err != nil {
    tx.Rollback()
    return err
}

if err := tx.Update(&user); err != nil {
    tx.Rollback()
    return err
}

return tx.Commit()
```

---

## 🔌 Repository Integration

### Adapter

```
func (r *authReader) Adapter() *orm.SqlQueryAdapter {
    return orm.NewSqlQueryAdapter(r.ctx, r.Db, dialect, cfg)
}
```

### Usage

```
db := r.Adapter()

err := db.UseModel(User{}).
    Where("id = ?", 1).
    Scan(&user)
```

---

## 🧠 Context Handling

- Context injected from service layer
- Stored inside QueryBuilder
- No need to pass context manually

---

## 🪵 Logging

```
o.SetLogger(logger.DefaultLogger{}, true)
```

---

## ⚡ Performance

- Metadata caching (no repeated reflection)
- Fast scan via column index mapping

---

## 🧩 Dialect Support

| Dialect    | Placeholder | RETURNING |
|------------|------------|----------|
| PostgreSQL | $1         | Yes      |
| MySQL      | ?          | No       |
| Oracle     | :1         | Yes      |

---

## 📌 Summary

This ORM focuses on:

- simplicity
- performance
- flexibility
- production readiness
