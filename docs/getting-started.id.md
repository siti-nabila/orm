# Memulai

## Instalasi

```bash
go get github.com/siti-nabila/orm
```

## Import

```go
import (
    "context"
    "database/sql"

    "github.com/siti-nabila/orm/config"
    "github.com/siti-nabila/orm/dialect"
    "github.com/siti-nabila/orm/orm"
)
```

## Memilih Dialek

Repositori ini menyediakan constructor untuk dialek yang didukung:

```go
pg := dialect.NewPostgres()
my := dialect.NewMysql()
ora := dialect.NewOracle()
```

Gunakan salah satunya saat membuat adapter.

## Adapter Query

Gunakan `NewSqlQueryAdapter` untuk alur baca/query:

```go
func NewUserReader(ctx context.Context, conn *sql.DB) *orm.SqlQueryAdapter {
    return orm.NewSqlQueryAdapter(ctx, conn, dialect.NewPostgres(), config.Config{})
}
```

Adapter query menyediakan:

- `UseModel(model any)`
- `SetLogger(logger.Logger, debug bool)`

## Adapter Transaksi

Gunakan `NewSqlTransactionAdapter` ketika Anda sudah memiliki `*sql.Tx` dan ingin
melakukan penulisan atau penguncian pada tingkat transaksi:

```go
sqlTx, err := conn.BeginTx(ctx, nil)
if err != nil {
    return err
}

tx := orm.NewSqlTransactionAdapter(ctx, sqlTx, dialect.NewPostgres(), config.Config{})
```

Pemanggil menentukan kapan melakukan commit atau rollback. Method penulisan ORM
tidak melakukan commit atau rollback secara otomatis.

## Tag Model

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

Jika `TableName()` tidak diimplementasikan, nama tabel berasal dari tipe struct.
Ketika `config.Config{UseSnakeCase: true}` diaktifkan, nama default tabel dan
kolom menggunakan snake_case.
