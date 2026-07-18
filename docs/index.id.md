# Dokumentasi ORM

Dokumentasi ini menjelaskan API publik yang tersedia dalam repositori ini.
ORM ini sengaja dibuat kecil: SQL ditulis secara eksplisit, kepemilikan transaksi
tetap berada pada pemanggil, dan perilaku setiap dialek tetap terlihat jelas.

## Panduan

- [Memulai](getting-started.id.md)
- [Query Builder](query-builder.id.md)
- [Create, Update, dan Penulisan Massal](create-update.id.md)
- [Memindai Hasil](scan.id.md)
- [Paginasi](pagination.id.md)
- [Dry Run](dry-run.id.md)
- [Transaksi dan Penguncian](transactions.id.md)
- [Catatan Dialek](dialects.id.md)
- [Contoh](examples.id.md)

## Model yang Digunakan dalam Contoh

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

Field struct dipetakan dengan tag `sql`. Gunakan `column:name` untuk menentukan
nama kolom database, `primaryKey` untuk menandai field primary key, dan `where`
untuk menandai kondisi update cadangan bagi model tanpa primary key. Gunakan
`sql:"-"` untuk mengabaikan suatu field.

```go
type ProfileUpdate struct {
    UserID      uint64 `sql:"column:user_id;where"`
    DisplayName string `sql:"column:display_name"`
}
```

Lihat [Create, Update, dan Penulisan Massal](create-update.id.md) untuk prioritas
kondisi, perilaku zero value, update berantai, dan aturan keamanan.
