# Create, Update, dan Penulisan Massal

API penulisan tersedia pada `*orm.ORM` dan `*orm.SqlTransactionAdapter`. Adapter
transaksi adalah entry point pengguna yang paling langsung ketika Anda sudah
memiliki `*sql.Tx`.

## Create

```go
tx := orm.NewSqlTransactionAdapter(ctx, sqlTx, dialect.NewPostgres(), config.Config{})

user := User{
    Name:   "joko",
    Email:  "joko@example.com",
    Status: "ACTIVE",
}

if err := tx.Create(&user); err != nil {
    tx.Rollback()
    return err
}

return tx.Commit()
```

`Create` mengharapkan pointer ke struct. Jika dialek mendukung returning dan
primary key dipetakan, primary key yang dihasilkan dapat dipindai kembali ke model.

## Update

Update menerima pointer ke struct. Tanpa map field eksplisit, update dibangun
dari nilai struct yang dipetakan dan primary key.

```go
user.Email = "new@example.com"

if err := tx.Update(&user); err != nil {
    tx.Rollback()
    return err
}
```

Anda dapat meneruskan map untuk meng-update kolom tertentu:

```go
err := tx.Update(&user, map[string]any{
    "email":  "new@example.com",
    "status": "ACTIVE",
})
```

Key map merupakan nama kolom.

### Update Model Tanpa Primary Key

Model khusus update dapat menandai satu atau beberapa kolom sebagai kondisi
update dengan tag `where`. Ini berguna untuk tabel atau model penulisan yang
tidak memiliki primary key.

```go
type ProfileUpdate struct {
    TenantID    uint64 `sql:"column:tenant_id;where"`
    UserID      uint64 `sql:"column:user_id;where"`
    DisplayName string `sql:"column:display_name"`
}

func (ProfileUpdate) TableName() string {
    return "user_profiles"
}

update := &ProfileUpdate{
    TenantID:    7,
    UserID:      42,
    DisplayName: "Nabila",
}

if err := tx.Update(update); err != nil {
    tx.Rollback()
    return err
}
```

Kondisi yang dihasilkan menggabungkan semua kolom bertag dengan `AND`:

```sql
UPDATE user_profiles
SET display_name = $1
WHERE tenant_id = $2 AND user_id = $3
```

Argumen tetap berupa parameter dan diurutkan sebagai:

```go
[]any{"Nabila", uint64(7), uint64(42)}
```

Jika model memiliki `primaryKey`, primary key tetap menjadi sumber kondisi untuk
`Update`; kolom bertag `where` hanya menjadi cadangan ketika primary key tidak
didefinisikan. Pointer nil pada kondisi bertag ditolak. Zero value scalar seperti
`0`, `false`, dan `""` tetap merupakan nilai kondisi yang valid, sehingga pemanggil
harus menetapkannya dengan sengaja.

Saat meneruskan map update, kolom bertag `where` tidak boleh sekaligus di-update:

```go
// Ditolak karena user_id merupakan bagian dari kondisi update.
err := tx.Update(update, map[string]any{
    "user_id": uint64(84),
})
```

### Update Model Berantai

`UseModel` tersedia pada adapter transaksi untuk update berantai. Model harus
berupa pointer non-nil ke struct.

```go
result, err := tx.
    UseModel(update).
    Where("tenant_id = ?", tenantID).
    Updates()
if err != nil {
    tx.Rollback()
    return err
}

rows, err := result.RowsAffected()
```

`Updates()` menyertakan setiap field dengan tag `sql` eksplisit dalam `SET`,
termasuk zero value, kecuali field bertag `primaryKey` atau `where`. Gunakan
struct kecil yang khusus untuk update agar field zero value yang tidak terkait
tidak ikut ditulis.

Aturan kondisinya adalah:

1. Kondisi berantai eksplisit (`Where`, `OrWhere`, `WhereIn`, kondisi berkelompok,
   dan variannya) menjadi keseluruhan sumber `WHERE`.
2. Tanpa kondisi eksplisit, primary key non-zero digunakan.
3. Tanpa primary key, kolom bertag `where` digunakan.
4. Update tanpa kondisi ditolak.

Kondisi eksplisit tidak otomatis digabungkan dengan primary key atau kondisi
bertag. Sertakan setiap batasan tenant atau kepemilikan yang diperlukan dalam chain.

Perilaku ini berlaku untuk PostgreSQL, MySQL, dan Oracle. Konfigurasi placeholder
dan pengutipan identifier yang ada tetap berlaku.

### Logging Update

Logging query update melakukan dereference pointer scalar sebelum interpolasi
dan menggunakan nilai `database/sql/driver.Valuer` jika tersedia. Ini hanya
mengubah nilai yang ditampilkan dalam log; query dan argumen asli yang dikirim
ke database tidak berubah.

## Insert Lanjutan: CreateWith

`CreateWith` mengembalikan `CreateCommand` untuk returning column dan penanganan konflik.

```go
err := tx.CreateWith(&user).
    WithReturning("id", "email").
    Scan(&user)
```

Gunakan `ScanInto` ketika Anda menginginkan variabel tujuan eksplisit:

```go
var id int64
var email string

err := tx.CreateWith(&user).
    WithReturning("id", "email").
    ScanInto(&id, &email)
```

`Exec` digunakan untuk command create tanpa returning column:

```go
err := tx.CreateWith(&user).Exec()
```

Jika returning column dikonfigurasi, gunakan `Scan` atau `ScanInto`; `Exec`
menolak mode returning.

## Penanganan Konflik

`WithOnConflict` menerima `orm.OnConflict`.

```go
err := tx.CreateWith(&user).
    WithOnConflict(orm.OnConflict{
        TargetColumns: []string{"email"},
        DoUpdates:     []string{"name", "status"},
    }).
    Exec()
```

Assignment khusus tersedia melalui `orm.Value` dan `orm.Inc`:

```go
err := tx.CreateWith(&user).
    WithOnConflict(orm.OnConflict{
        TargetColumns: []string{"email"},
        Assignments: []orm.ConflictAssignment{
            {Column: "status", Expr: orm.Value("ACTIVE")},
            {Column: "login_count", Expr: orm.Inc("login_count", 1)},
        },
    }).
    Exec()
```

`DoNothing` juga didukung:

```go
err := tx.CreateWith(&user).
    WithOnConflict(orm.OnConflict{
        TargetColumns: []string{"email"},
        DoNothing:     true,
    }).
    Exec()
```

Jangan gabungkan `DoNothing` dengan assignment update.

## Bulk Insert

```go
users := []User{
    {Name: "A", Email: "a@example.com"},
    {Name: "B", Email: "b@example.com"},
}

err := tx.CreateBulk(users)
```

Bulk insert menerima slice struct atau pointer ke struct. Setiap baris harus
merujuk pada tabel dan layout yang kompatibel. PostgreSQL menggunakan perilaku
returning untuk primary key; MySQL dan Oracle mengeksekusi statement massal.

## Bulk Update

```go
err := tx.UpdateBulk(users)
```

Bulk update memerlukan primary key agar setiap baris dapat dicocokkan. Adapter
transaksi juga menyediakan `DryRunUpdateBulk`.

## Belum Diimplementasikan

API delete bukan bagian dari repositori ini.
