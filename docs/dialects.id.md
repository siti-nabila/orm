# Catatan Dialek

Constructor dialek yang didukung adalah:

```go
dialect.NewPostgres()
dialect.NewMysql()
dialect.NewOracle()
```

## PostgreSQL

- Nama dialek: `postgres`
- Placeholder: `$1`, `$2`, ...
- Paginasi: `LIMIT n OFFSET m`
- Pengutipan identifier mengembalikan identifier tanpa perubahan pada dialek saat ini.
- `SupportReturning()` bernilai `true`.
- Advisory lock: `pg_try_advisory_xact_lock($1)`.
- Bulk insert menggunakan mode query untuk memindai primary key yang dikembalikan.

### Argumen Array PostgreSQL

Executor DB dan transaksi menormalisasi argumen slice dan array non-byte dengan
`pq.Array` sebelum memanggil `database/sql`. Ini mendukung parameter array
PostgreSQL seperti `[]int64`, `[]uint32`, dan `[]float64` tanpa mengharuskan
pemanggil membungkus setiap nilai secara manual.

```go
_, err := executor.ExecContext(
    ctx,
    "UPDATE users SET role_ids = $1 WHERE id = $2",
    []int64{10, 20},
    userID,
)
```

Argumen `driver.Valuer` yang ada dipertahankan, `[]byte` tetap menjadi argumen
biner, dan pointer nil diteruskan sebagai `NULL`. Normalisasi ini hanya diterapkan
pada PostgreSQL; argumen MySQL dan Oracle mempertahankan perilaku sebelumnya.

## MySQL

- Nama dialek: `mysql`
- Placeholder: `?`
- Paginasi: `LIMIT n OFFSET m`
- Offset tanpa limit menggunakan `LIMIT 18446744073709551615 OFFSET n`.
- Pengutipan identifier menggunakan backtick.
- `SupportReturning()` bernilai `false`.
- Advisory lock: `GET_LOCK(?, 0)` and `RELEASE_LOCK(?)`.
- Advanced insert returning menggunakan pemindaian lanjutan khusus MySQL dalam
  implementasi.

## Oracle

- Nama dialek: `oracle`
- Placeholder: `:1`, `:2`, ...
- Paginasi: `OFFSET n ROWS FETCH NEXT m ROWS ONLY`.
- Pengutipan identifier menggunakan tanda kutip ganda.
- `SupportReturning()` bernilai `true`.
- Advisory lock menggunakan `DBMS_LOCK`.
- Pembungkus count menggunakan sintaks alias subquery tanpa `AS`.

## Mode Placeholder

`config.PlaceholderAuto` menggunakan placeholder bergaya nama untuk Oracle dan
placeholder bernomor untuk dialek lain yang didukung. Rebinding query builder
menjaga placeholder kondisi mentah `?` agar sesuai dengan setiap dialek.

## Nilai dalam Log

Logger default menampilkan slice dan array PostgreSQL sebagai `ARRAY[...]`.
MySQL dan Oracle mempertahankan tampilan slice dalam tanda kurung. Nilai scalar,
slice, dan array yang panjang dibatasi hingga 120 byte dengan
`...(truncated)...` di tengah, sambil mempertahankan bagian awal dan akhirnya.
Ini hanya memengaruhi output log; SQL yang dieksekusi dan nilai parameter tidak berubah.
