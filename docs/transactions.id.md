# Transaksi dan Penguncian

## Kepemilikan Transaksi

Pemanggil memiliki batas transaksi. Method penulisan ORM tidak melakukan commit
atau rollback otomatis. Anda menentukan kapan memanggil `Commit` atau `Rollback`.

```go
sqlTx, err := conn.BeginTx(ctx, nil)
if err != nil {
    return err
}

tx := orm.NewSqlTransactionAdapter(ctx, sqlTx, dialect.NewPostgres(), config.Config{})

if err := tx.Create(&user); err != nil {
    tx.Rollback()
    return err
}

return tx.Commit()
```

`SqlTransactionAdapter` menyediakan:

- `Create`
- `Update`
- `CreateBulk`
- `UpdateBulk`
- `DryRunUpdateBulk`
- `CreateWith`
- `UseModel`
- `TryLock`
- `Commit`
- `Rollback`
- `SetLogger`

`Commit` dan `Rollback` melepaskan lock yang diperoleh sebelum mengakhiri transaksi.

`UseModel` mempertahankan context dan executor transaksi sambil menyediakan query
builder. Method ini dapat digunakan untuk update model berantai:

```go
result, err := tx.
    UseModel(&ProfileUpdate{
        TenantID:    7,
        UserID:      42,
        DisplayName: "Nabila",
    }).
    Updates()
if err != nil {
    tx.Rollback()
    return err
}

if _, err := result.RowsAffected(); err != nil {
    tx.Rollback()
    return err
}

return tx.Commit()
```

`Updates` tidak melakukan commit atau rollback secara otomatis. Kepemilikan
transaksi tetap berada pada pemanggil.

## Advisory Lock

`TryLock(ctx, key)` bersifat non-blocking. Method ini menormalisasi key, memeriksa
apakah adapter yang sama sudah memperolehnya, lalu mendelegasikannya ke dialek.

```go
locked, err := tx.TryLock(ctx, "user:123")
if err != nil {
    tx.Rollback()
    return err
}
if !locked {
    tx.Rollback()
    return fmt.Errorf("lock not acquired")
}
```

### PostgreSQL

PostgreSQL menggunakan `pg_try_advisory_xact_lock($1)` dengan hash dari lock key.
Lock memiliki cakupan transaksi, sehingga query pelepasan eksplisit tidak diperlukan.

### MySQL

MySQL menggunakan `GET_LOCK(?, 0)` dan melepaskan lock yang diperoleh dengan
`RELEASE_LOCK(?)` saat `Commit` atau `Rollback`.

### Oracle

Oracle menggunakan `DBMS_LOCK.ALLOCATE_UNIQUE` dan `DBMS_LOCK.REQUEST` dengan
`release_on_commit => TRUE`. Implementasi menganggap return code `0` sebagai
berhasil diperoleh dan `1` sebagai tidak diperoleh.

## Logging Lock

Atur `config.Config{LogLockQuery: true}` untuk mencatat query lock melalui logger
yang telah dikonfigurasi.
