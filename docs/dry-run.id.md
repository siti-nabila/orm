# Dry Run

Method dry run mengembalikan SQL, argumen, dan mode tanpa mengeksekusi query.

Bentuk hasilnya adalah:

```go
type DryRunResult struct {
    Query string
    Args  []any
    Mode  builder.DryRunMode
}
```

Mode yang tersedia:

- `builder.DryRunModeExec`
- `builder.DryRunModeQuery`
- `builder.DryRunModeQueryRow`

## Create dan Update

`*orm.ORM` menyediakan:

```go
result, err := core.DryRunCreate(&user)
result, err := core.DryRunUpdate(&user)
result, err := core.DryRunUpdate(&user, map[string]any{"email": "new@example.com"})
result, err := core.DryRunUpdateBulk(users)
```

Saat menggunakan `CreateWith`, dry run tersedia pada command:

```go
result, err := tx.CreateWith(&user).
    WithReturning("id").
    DryRun()
```

Adapter transaksi saat ini menyediakan `DryRunUpdateBulk`.

### Dry Run untuk Update Model Berantai

Gunakan `DryRunUpdates` untuk memeriksa update model berantai tanpa mengeksekusinya:

```go
update := &ProfileUpdate{
    TenantID:    7,
    UserID:      42,
    DisplayName: "Nabila",
}

result, err := tx.
    UseModel(update).
    DryRunUpdates()
if err != nil {
    return err
}

fmt.Println(result.Query)
fmt.Println(result.Args)
fmt.Println(result.Mode)
```

Untuk model yang didokumentasikan dalam [Create, Update, dan Penulisan Massal](create-update.id.md),
verifikasi bentuk output berikut:

| Dialek | Fragmen query penting |
| --- | --- |
| PostgreSQL | `SET display_name = $1 WHERE tenant_id = $2 AND user_id = $3` |
| MySQL | `SET display_name = ? WHERE tenant_id = ? AND user_id = ?` |
| Oracle | `SET display_name = :display_name WHERE tenant_id = :tenant_id AND user_id = :user_id` |

Untuk ketiga dialek:

```go
result.Args == []any{"Nabila", uint64(7), uint64(42)}
result.Mode == builder.DryRunModeExec
```

`DryRunUpdates` menggunakan builder dan pemeriksaan keamanan yang sama dengan
`Updates`. Method ini menolak update tanpa kondisi dan tidak mengeksekusi,
melakukan commit, atau melakukan rollback transaksi.

## Query Builder

```go
result, err := db.UseModel(User{}).
    Where("status = ?", "ACTIVE").
    OrderBy("id ASC").
    DryRun()
```

Dry run untuk count dan first:

```go
countResult, err := db.UseModel(User{}).
    Where("status = ?", "ACTIVE").
    DryRunCount()

firstResult, err := db.UseModel(User{}).
    Where("email = ?", email).
    DryRunFirst()
```

## Dry Run Paginator

`DryRunScanPaginate` menyediakan kedua query yang digunakan oleh `ScanPaginate`.

```go
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

fmt.Println(result.Count.Query)
fmt.Println(result.Count.Args)
fmt.Println(result.Count.Mode)

fmt.Println(result.Data.Query)
fmt.Println(result.Data.Args)
fmt.Println(result.Data.Mode)
```

`DryRunPaginate` adalah method dry run untuk API `Paginate` lama yang berbasis
`pagination.Params`.

## Logging Dry Run

Atur `config.Config{LogDryRunQuery: true}` dan konfigurasikan logger untuk
menghasilkan log dry run. Adapter query default memasang `logger.DefaultLogger`
menggunakan `EnableDebug`; logging dry run tetap dikendalikan oleh `LogDryRunQuery`.

Nilai panjang dalam log default yang ditampilkan akan dipotong, dan argumen array
PostgreSQL ditampilkan sebagai `ARRAY[...]`. `DryRunResult.Query`,
`DryRunResult.Args`, dan `DryRunResult.Mode` tidak diubah oleh format log. Secara
khusus, argumen yang dikembalikan DryRun tetap merupakan nilai parameter asli
yang diberikan pemanggil.
