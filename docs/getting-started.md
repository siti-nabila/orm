# Getting Started

## Install

```bash
go get github.com/siti-nabila/orm
```

## Imports

```go
import (
    "context"
    "database/sql"

    "github.com/siti-nabila/orm/config"
    "github.com/siti-nabila/orm/dialect"
    "github.com/siti-nabila/orm/orm"
)
```

## Choose A Dialect

The repository includes constructors for the supported dialects:

```go
pg := dialect.NewPostgres()
my := dialect.NewMysql()
ora := dialect.NewOracle()
```

Use one of those when creating an adapter.

## Query Adapter

Use `NewSqlQueryAdapter` for read/query flows:

```go
func NewUserReader(ctx context.Context, conn *sql.DB) *orm.SqlQueryAdapter {
    return orm.NewSqlQueryAdapter(ctx, conn, dialect.NewPostgres(), config.Config{})
}
```

The query adapter exposes:

- `UseModel(model any)`
- `SetLogger(logger.Logger, debug bool)`

## Transaction Adapter

Use `NewSqlTransactionAdapter` when you already have a `*sql.Tx` and want to
perform writes or transaction-level locks:

```go
sqlTx, err := conn.BeginTx(ctx, nil)
if err != nil {
    return err
}

tx := orm.NewSqlTransactionAdapter(ctx, sqlTx, dialect.NewPostgres(), config.Config{})
```

The caller decides when to commit or roll back. ORM write methods do not commit
or roll back automatically.

## Model Tags

```go
type User struct {
    ID        int64  `sql:"column:id;primaryKey"`
    Name      string `sql:"column:name"`
    Email     string `sql:"column:email"`
    Status    string `sql:"column:status"`
    CreatedAt string `sql:"column:created_at"`
}

func (User) TableName() string {
    return "users"
}
```

If `TableName()` is not implemented, table names come from the struct type.
When `config.Config{UseSnakeCase: true}` is enabled, default table and column
names use snake_case.
