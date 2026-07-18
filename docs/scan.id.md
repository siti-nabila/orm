# Memindai Hasil

`Scan` dan `First` adalah method query builder.

## Memindai ke Struct

```go
var user User

err := db.UseModel(User{}).
    Where("email = ?", email).
    First(&user)
```

`First` menerapkan `LIMIT 1` dan memindai satu baris. Baris yang tidak ditemukan
dinormalisasi melalui lapisan normalisasi error repositori.

## Memindai ke Slice Struct

```go
var users []User

err := db.UseModel(User{}).
    Where("status = ?", "ACTIVE").
    OrderBy("created_at DESC").
    Scan(&users)
```

Tujuan harus berupa pointer ke slice.

## Memindai Nilai Primitif

Pemindaian nilai primitif didukung ketika query mengembalikan tepat satu kolom.

```go
var emails []string

err := db.UseModel(User{}).
    Select("email").
    Where("status = ?", "ACTIVE").
    Scan(&emails)
```

Untuk satu nilai primitif:

```go
var total int64

err := db.UseModel(User{}).
    Select("COUNT(*)").
    First(&total)
```

Pemindaian primitif memerlukan satu kolom yang dipilih.

## Penanganan Pemindaian Khusus Dialek

Scanner memiliki penanganan bersama untuk integer, unsigned integer, float,
boolean, `[]byte`, dan field pointer string nullable bagi nilai driver MySQL dan
Oracle yang menyerupai angka atau string.

PostgreSQL memiliki penanganan tambahan untuk field array, termasuk `[]string`,
`[]int`, `[]int64`, `[]uint32`, dan `[]uint64`.

Implementasi juga melakukan pemeriksaan overflow pada konversi numerik.
