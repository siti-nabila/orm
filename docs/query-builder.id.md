# Query Builder

Entry point utama untuk query baca adalah:

```go
q := db.UseModel(User{})
```

`UseModel` membuat query builder dan menetapkan tabel model. Method tingkat
rendah `Table(model)` juga tersedia pada `query.QueryBuilder`; `UseModel`
memanggilnya untuk Anda.

## Select

Tanpa `Select`, builder memilih kolom model yang telah dipetakan.

```go
err := db.UseModel(User{}).
    Select("id", "email").
    Scan(&users)
```

`Select` juga menerima ekspresi select mentah ketika string berisi spasi, titik,
tanda kurung, `*`, atau ` AS `:

```go
err := db.UseModel(User{}).
    Select("status", "COUNT(*) AS total").
    GroupBy("status").
    Scan(&rows)
```

Tidak ada method publik `SelectExpr` terpisah dalam implementasi saat ini.

## Where

`Where` dan `OrWhere` menerima fragmen SQL dengan placeholder `?`:

```go
err := db.UseModel(User{}).
    Where("status = ?", "ACTIVE").
    OrWhere("email LIKE ?", "%@example.com").
    Scan(&users)
```

Kondisi berkelompok didukung:

```go
err := db.UseModel(User{}).
    Where("status = ?", "ACTIVE").
    WhereGroup(func(q *query.QueryBuilder) {
        q.Where("email LIKE ?", "%@example.com").
            OrWhere("email LIKE ?", "%@company.com")
    }).
    Scan(&users)
```

Gunakan `OrWhereGroup` ketika kelompok harus dihubungkan dengan `OR`.

## Operator Where yang Aman

`WhereOp` dan `OrWhereOp` menyediakan operator bertipe sambil mempertahankan API
`Where` yang sudah ada.

```go
builder, err := db.UseModel(User{}).
    WhereOp("status", orm.OpEqual, "ACTIVE")
if err != nil {
    return err
}

builder, err = builder.OrWhereOp("email", orm.OpLike, "%@example.com")
if err != nil {
    return err
}

err = builder.Scan(&users)
```

Operator yang tersedia:

- `orm.OpEqual`
- `orm.OpNotEqual`
- `orm.OpLessThan`
- `orm.OpLessThanEqual`
- `orm.OpGreaterThan`
- `orm.OpGreaterThanEqual`
- `orm.OpLike`
- `orm.OpNotLike`

`Where` tetap didukung dan tidak deprecated. Gunakan `WhereOp` ketika Anda
menginginkan konstanta operator bertipe.

## Where Full Text PostgreSQL

`WhereFullText` dan `OrWhereFullText` menambahkan kondisi pencarian full text
khusus PostgreSQL untuk kolom `tsvector`:

```go
builder, err := db.UseModel(Profile{}).
    Join("user_profile_search ups", "ups.profile_id = profiles.id").
    WhereFullText("ups.fts_keyword", keyword)
if err != nil {
    return err
}
```

Kondisi yang dihasilkan adalah:

```sql
ups.fts_keyword @@ websearch_to_tsquery('simple', ?)
```

Placeholder di-rebind berdasarkan dialek, sehingga output DryRun PostgreSQL
menggunakan `$1`. MySQL dan Oracle mengembalikan
`ErrUnsupportedSearchModeForDialect` untuk helper ini. Gunakan `WhereOp` hanya
untuk operator perbandingan biasa; pencarian full text tidak direpresentasikan
sebagai `=`, `<`, `LIKE`, atau `IN`.

## Daftar IN

```go
err := db.UseModel(User{}).
    WhereIn("status", []string{"ACTIVE", "PENDING"}).
    WhereNotIn("email", []string{"blocked@example.com"}).
    Scan(&users)
```

`OrWhereIn` dan `OrWhereNotIn` juga tersedia.

## Join

Builder mendukung inner, left, dan right join:

```go
err := db.UseModel(User{}).
    Join("roles", "roles.user_id = users.id").
    LeftJoin("profiles", "profiles.user_id = users.id").
    RightJoin("accounts", "accounts.user_id = users.id").
    Scan(&users)
```

Tabel join dan ekspresi `ON` ditulis sebagai string eksplisit.

## Pengurutan dan Batas

```go
err := db.UseModel(User{}).
    OrderBy("created_at DESC").
    Limit(20).
    Offset(40).
    Scan(&users)
```

`Limit` mengabaikan nilai yang kurang dari atau sama dengan nol. `Offset`
mengabaikan nilai negatif.

## Distinct, Group By, dan Having

```go
err := db.UseModel(User{}).
    Select("status", "COUNT(*) AS total").
    Distinct().
    GroupBy("status").
    Having("COUNT(*) > ?", 1).
    OrHaving("COUNT(*) = ?", 0).
    Scan(&rows)
```

Query count paginator menggunakan query count terbungkus untuk bentuk kompleks
seperti `DISTINCT`, `GROUP BY`, `HAVING`, ekspresi select mentah, dan select alias.

## First

`First` menerapkan `LIMIT 1`, menjalankan query, dan memindai satu baris ke tujuan:

```go
var user User
err := db.UseModel(User{}).
    Where("email = ?", email).
    First(&user)
```

## Komposisi subquery

Susun child builder melalui transaction yang sama. Child disimpan sebagai
snapshot: perubahan builder asal sesudah komposisi tidak mengubah parent.
Dialect, placeholder, identifier quoting dan urutan argument mengikuti parent,
termasuk offset setelah nilai SET pada UPDATE. Child nil/invalid atau alias
invalid menghasilkan error sebelum eksekusi. Alias derived table harus berupa
identifier sederhana dan tidak memakai AS.

```go
eligible := tx.UseModel(UserRole{}).
    Select("user_id").Where("status = ?", "ACTIVE")

q := tx.UseModel(User{}).
    WhereExists(tx.UseModel(UserRole{}).
        Select("user_id").Where("user_id = users.id").Where("status = ?", "ACTIVE"))

q = q.WhereInQuery("users.id", eligible)
q = q.WhereNotExists(tx.UseModel(UserRole{}).
    Select("user_id").Where("user_id = users.id").Where("status = ?", "REVOKED"))

names := tx.UseModel(UserRole{}).
    Select("user_id", "MIN(role_name) AS first_name").
    Where("status = ?", "ACTIVE").GroupBy("user_id")
q = q.LeftJoinQuery(names, "role_names", "role_names.user_id = users.id")

preview, err := q.DryRun()
count, err := q.DryRunCount()
```

DryRun, count dan pagination tidak mengubah builder asal. WhereExists,
WhereNotExists dan WhereInQuery juga tersedia pada Updates/DryRunUpdates.
Column, projection, correlation dan ON tetap berupa expression tepercaya seperti
Where existing; whitelist identifier dan bind nilai request sebagai argument.
Caller tidak perlu menulis SELECT utuh atau mengganti placeholder sendiri.
