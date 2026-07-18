# Pencarian PostgreSQL Full Text Trigram

## Overview

`orm.SearchModeFullTextTrigram` menggabungkan dua kondisi PostgreSQL dengan
`OR`:

- pencarian full-text prefix pada kolom `tsvector`, menggunakan
  `to_tsquery`;
- fallback contains yang case-insensitive pada kolom teks yang dikonfigurasi,
  menggunakan `ILIKE`.

Nama mode ini merujuk pada kemungkinan mengoptimalkan fallback `ILIKE` dengan
index `pg_trgm`. Implementasi ORM tidak memakai operator similarity (`%`),
fungsi `similarity`, atau pencocokan typo. Hasil fallback tetap ditentukan oleh
`ILIKE '%keyword%'`.

Dokumen ini mengikuti API ORM yang digunakan oleh `grpc-auth` pada
`github.com/siti-nabila/orm v1.7.0` dan implementasi consumer tersebut.

Perbedaan mode pencarian aktual:

| Mode | Kondisi | Dialek | Catatan |
| --- | --- | --- | --- |
| `SearchModeContains` | `Column LIKE ?`, argumen `%keyword%` | PostgreSQL, MySQL, Oracle | Mode default ketika `Mode` kosong; case sensitivity mengikuti database/collation. |
| `SearchModePrefix` | `Column LIKE ?`, argumen `keyword%` | PostgreSQL, MySQL, Oracle | Prefix teks biasa, bukan full-text. |
| `SearchModeFullText` | `FullTextColumn <operator> websearch_to_tsquery('<language>', ?)` | PostgreSQL | Operator default adalah `@@`; keyword diteruskan sebagai argumen tanpa transformasi prefix. |
| `SearchModeTrigram` | `Column ILIKE ?`, argumen `%keyword%` | PostgreSQL | Tidak menghitung similarity dan tidak fuzzy; nama mode menunjukkan bahwa `pg_trgm` dapat mengindeks pola ini. |
| `SearchModeFullTextTrigram` | `FullTextColumn @@ to_tsquery(...) OR Column ILIKE ...` | PostgreSQL | Full-text prefix ditambah fallback contains; bagian full-text selalu memakai `@@` pada implementasi saat ini. |

## Kapan Menggunakan Full Text Trigram

Gunakan mode ini ketika aplikasi membutuhkan prefix per token—misalnya `nab`
menemukan lexeme yang berawalan `nab`—serta fallback substring pada search
document teks. Mode ini cocok ketika:

- search document full-text sudah disimpan sebagai `tsvector`;
- consumer dapat menyediakan kolom teks fallback yang merepresentasikan data
  yang ingin dicari;
- PostgreSQL adalah database yang digunakan;
- pencarian typo atau ranking similarity bukan kebutuhan. Jika keduanya
  dibutuhkan, implementasi ini belum menyediakannya.

## Request

Contoh utama request `ListUsers` pada `grpc-auth`:

```json
{
  "query": {
    "page": 1,
    "limit": 2,
    "sort": [
      {
        "field": "auth_id",
        "desc": false
      }
    ],
    "search": {
      "fields": ["keyword"],
      "keyword": "nabila",
      "mode": "SEARCH_MODE_FULL_TEXT_TRIGRAM"
    },
    "last_id": "0"
  }
}
```

- `page` adalah nomor halaman berbasis 1. `grpc-auth` mempertahankannya sebagai
  nomor halaman untuk offset di dalam batch cursor.
- `limit` adalah jumlah item yang ditampilkan. Nilai `0` atau negatif
  dinormalisasi consumer ke default 10; nilai di atas maksimum dibatasi ke
  `pagination.MaxLimit` (1000 pada versi ini).
- `sort` berisi urutan hasil. `field: "auth_id"` dipetakan ke `a.id`, dan
  `desc: false` menghasilkan `ASC`. Pagination cursor user di `grpc-auth` hanya
  menerima sort `auth_id`.
- `search.fields` memilih alias pencarian publik. `keyword` di sini adalah key
  pada `SearchFields`; ini **bukan** nama kolom database.
- `search.keyword` adalah teks pencarian pengguna.
- `search.mode` memilih enum protobuf yang dipetakan ke mode ORM.
- `last_id` adalah cursor batch. String `"0"` (atau string kosong) menandakan
  batch awal; nilai nonzero harus berupa integer positif.

## Mapping Request ke ORM

Handler `grpc-auth` memetakan enum secara eksplisit:

```go
func searchModeFromProto(mode paginator.SearchMode) (orm.SearchMode, error) {
    switch mode {
    case paginator.SearchMode_SEARCH_MODE_UNSPECIFIED:
        return "", nil
    case paginator.SearchMode_SEARCH_MODE_CONTAINS:
        return orm.SearchModeContains, nil
    case paginator.SearchMode_SEARCH_MODE_PREFIX:
        return orm.SearchModePrefix, nil
    case paginator.SearchMode_SEARCH_MODE_FULL_TEXT:
        return orm.SearchModeFullText, nil
    case paginator.SearchMode_SEARCH_MODE_TRIGRAM:
        return orm.SearchModeTrigram, nil
    case paginator.SearchMode_SEARCH_MODE_FULL_TEXT_TRIGRAM:
        return orm.SearchModeFullTextTrigram, nil
    default:
        return "", ormdictionary.ErrInvalidSearchMode
    }
}
```

Snippet tersebut memakai import aktual berikut:

```go
import (
    "github.com/siti-nabila/grpc-auth/pb/paginator"
    "github.com/siti-nabila/orm/orm"
    ormdictionary "github.com/siti-nabila/orm/pkg/dictionary"
)
```

Setelah `PageQuery` dipetakan oleh handler, request utama menghasilkan:

```go
orm.QueryOptions{
    Page:  1,
    Limit: 2,
    Sort: []orm.SortField{
        {
            Field: "auth_id",
            Desc:  false,
        },
    },
    Search: &orm.SearchQuery{
        Fields:  []string{"keyword"},
        Keyword: "nabila",
        Mode:    orm.SearchModeFullTextTrigram,
    },
}
```

`last_id` tidak menjadi field `QueryOptions` pada tahap handler. Service
`grpc-auth` memanggil `paginator.Build` dalam `ModeCursor`, menghapus sort dari
request lalu memasangnya kembali sebagai default sort yang tervalidasi, dan
menambahkan opsi berikut sebelum memanggil repository:

```go
opts.InMemoryOffset = &orm.InMemoryOffsetOptions{
    Cursor: orm.Cursor{
        Field: "auth_id",
        Value: "", // last_id "0" dinormalisasi menjadi cursor awal kosong
    },
    MaxLimit: pagination.MaxLimit,
}
```

Untuk `last_id: "18"`, parser consumer menghasilkan `Value: int64(18)`.

## Konfigurasi Consumer

`grpc-auth` memisahkan allowlist field umum dari konfigurasi search alias:

```go
allowedFields := map[string]string{
    "auth_id": "a.id",
    "email":   "a.email",
    "name":    `p."name"`,
    "phone":   "p.phone",
}

searchFields := map[string]orm.SearchFieldConfig{
    "keyword": {
        Column:           "ups.fts_lexeme_text",
        FullTextColumn:   "ups.fts_keyword",
        FullTextLanguage: orm.FullTextSimple,
        Modes: []orm.SearchMode{
            orm.SearchModeFullText,
            orm.SearchModeFullTextTrigram,
        },
    },
}
```

- `Column` adalah ekspresi kolom teks untuk fallback. Pada mode ini ORM
  membentuk `Column ILIKE '%' || lower($n) || '%'`.
- `FullTextColumn` adalah ekspresi kolom `tsvector` untuk kondisi full-text.
- `FullTextLanguage` memilih konfigurasi text search. Nilai yang tersedia
  adalah `orm.FullTextSimple` dan `orm.FullTextEnglish`; nilai kosong default ke
  `simple`.
- `Modes` adalah allowlist mode untuk alias tersebut. Daftar kosong hanya
  mengizinkan mode portable (`contains` dan `prefix`), sehingga mode lanjutan
  harus dicantumkan secara eksplisit.

Key request dipetakan oleh aplikasi ke ekspresi database yang sudah ditentukan.
Karena itu client tidak dapat mengirim nama kolom, join alias, atau ekspresi SQL
secara langsung. `AllowedFields` digunakan antara lain untuk sort, sedangkan
`SearchFields` mengizinkan alias pencarian seperti `keyword`.

## Menjalankan Query

Model hasil aktual pada consumer:

```go
type UserSearchRow struct {
    AuthID  int64  `sql:"column:auth_id" json:"auth_id"`
    Email   string `sql:"column:email" json:"email"`
    Name    string `sql:"column:name" json:"name"`
    Address string `sql:"column:address" json:"address"`
    Phone   string `sql:"column:phone" json:"phone"`
}

func (UserSearchRow) TableName() string {
    return "profile p"
}
```

Query dasar consumer:

```go
query := db.UseModel(UserSearchRow{}).
    Select(
        "a.id AS auth_id",
        "a.email",
        `p."name"`,
        "p.address",
        "p.phone",
    ).
    Join("auth a", "a.id = p.user_id").
    Join("user_profile_search ups", "ups.profile_id = p.id")
```

Pemanggilan ORM lengkap:

```go
func searchUsers(
    ctx context.Context,
    db *orm.SqlQueryAdapter,
    opts orm.QueryOptions,
) (orm.PageData[UserSearchRow], error) {
    allowedFields := map[string]string{
        "auth_id": "a.id",
        "email":   "a.email",
        "name":    `p."name"`,
        "phone":   "p.phone",
    }
    searchFields := map[string]orm.SearchFieldConfig{
        "keyword": {
            Column:           "ups.fts_lexeme_text",
            FullTextColumn:   "ups.fts_keyword",
            FullTextLanguage: orm.FullTextSimple,
            Modes: []orm.SearchMode{
                orm.SearchModeFullText,
                orm.SearchModeFullTextTrigram,
            },
        },
    }

    query := db.UseModel(UserSearchRow{}).
        Select(
            "a.id AS auth_id",
            "a.email",
            `p."name"`,
            "p.address",
            "p.phone",
        ).
        Join("auth a", "a.id = p.user_id").
        Join("user_profile_search ups", "ups.profile_id = p.id")

    rows := make([]UserSearchRow, 0)
    pageData, err := orm.QueryPageWithConfig(
        ctx,
        query,
        &rows,
        orm.QueryPageConfig{
            AllowedFields: allowedFields,
            SearchFields:  searchFields,
        },
        opts,
    )
    if err != nil {
        return orm.PageData[UserSearchRow]{}, err
    }
    return pageData, nil
}
```

Import yang diperlukan:

```go
import (
    "context"

    "github.com/siti-nabila/orm/orm"
)
```

`QueryPageWithConfig` menerima context, query builder yang telah terikat model,
pointer ke slice tujuan, mapping/allowlist, dan `QueryOptions`. Fungsi ini
menjalankan count query dan data query, mengisi slice, lalu mengembalikan
`orm.PageData[T]`. `PageData` memuat `Items`, `Total`, `Page`, `Limit`,
`TotalPages`, `HasNext`, `HasPrev`, dan `NextCursor`. Error validasi, pembangunan
SQL, eksekusi database, atau scan dikembalikan tanpa ditelan; consumer dapat
memetakkannya ke status gRPC/API.

## Cara Kerja Pencarian

Keyword dipecah dengan `strings.Fields`. Setiap token mendapat suffix `:*` dan
digabung dengan ` & ` untuk argumen full-text. Whitespace fallback dinormalisasi
menjadi satu spasi:

```text
nabila       -> nabila:*
siti nabila  -> siti:* & nabila:*
```

Untuk request utama, kondisi parameterized PostgreSQL adalah:

```sql
WHERE (
    ups.fts_keyword @@ to_tsquery('simple', $1)
    OR ups.fts_lexeme_text ILIKE '%' || lower($2) || '%'
)
ORDER BY a.id ASC
LIMIT 2;
```

```go
args := []any{"nabila:*", "nabila"}
```

Ini adalah bentuk konseptual untuk `QueryOptions` langsung dengan ukuran page
2. Placeholder dan detail query final ditentukan dialect/query builder; jangan
menginterpolasi keyword ke SQL.

Pada jalur `grpc-auth`, service selalu menambahkan `InMemoryOffset` dengan
`MaxLimit: pagination.MaxLimit`. Karena itu data query batch awal secara aktual
berakhir dengan `LIMIT 1000` tanpa `OFFSET`, kemudian ORM mengambil dua item
untuk page 1 di memori. Count query terpisah menggunakan kondisi pencarian yang
sama tetapi tidak memiliki `ORDER BY`, `LIMIT`, atau `OFFSET`.

## Contoh Hasil

Data ilustratif:

| auth_id | email | name |
| ---: | --- | --- |
| 12 | nabila@example.com | Siti Nabila |
| 18 | nabilah@example.com | Nabilah Putri |
| 25 | budi@example.com | Budi Santoso |

Contoh respons consumer:

```json
{
  "items": [
    {
      "id": 12,
      "email": "nabila@example.com",
      "name": "Siti Nabila",
      "address": "Jakarta",
      "phone": "081234567890"
    },
    {
      "id": 18,
      "email": "nabilah@example.com",
      "name": "Nabilah Putri",
      "address": "Bandung",
      "phone": "081298765432"
    }
  ],
  "total": 2,
  "page": 1,
  "limit": 2,
  "total_pages": 1,
  "has_next": false,
  "has_prev": false,
  "next_cursor": "18"
}
```

Data dan seluruh metadata di atas hanya ilustrasi. Implementasi ORM mengambil
`NextCursor` dari field cursor pada item terakhir yang ditampilkan. Karena itu
cursor dapat terisi walaupun `HasNext` bernilai `false`; gunakan `HasNext` untuk
menentukan apakah hasil menurut total/page masih berlanjut.

## Cursor Pagination

`"last_id": "0"` berarti request batch awal. `grpc-auth` mengubahnya menjadi
cursor kosong, sehingga ORM tidak menambahkan kondisi `a.id > ...` atau
`a.id < ...`.

Jika respons menghasilkan `"next_cursor": "18"`, bentuk request yang diminta
untuk batch berikutnya adalah:

```json
{
  "query": {
    "page": 2,
    "limit": 2,
    "sort": [
      {
        "field": "auth_id",
        "desc": false
      }
    ],
    "search": {
      "fields": ["keyword"],
      "keyword": "nabila",
      "mode": "SEARCH_MODE_FULL_TEXT_TRIGRAM"
    },
    "last_id": "18"
  }
}
```

Perilaku aktual perlu diperhatikan: `grpc-auth` meneruskan `page: 2` dan cursor
18 bersamaan. Untuk sort ascending, ORM menambah `a.id > $n`, mengambil hingga
1000 baris tanpa SQL `OFFSET`, lalu menerapkan offset page 2 (`(2-1)*2`) di
memori. Artinya dua baris pertama setelah ID 18 dilewati. Jika yang diinginkan
adalah halaman pertama tepat setelah cursor 18, consumer harus mengirim
`page: 1`; jangan menganggap implementasi ini sebagai keyset pagination murni.
Untuk descending, operator cursor menjadi `<`.

Count query memakai kondisi cursor yang sama. Karena itu `Total`, `TotalPages`,
`HasNext`, dan `HasPrev` pada request dengan `last_id` nonzero dihitung terhadap
setelah filter cursor, bukan selalu terhadap keseluruhan hasil pencarian awal.

`NextCursor` diambil dari tag SQL `column:auth_id` pada item terakhir. Nilai
`last_id` nonzero divalidasi oleh `grpc-auth` sebagai integer positif sebelum
repository dipanggil.

## Persyaratan PostgreSQL dan Index

`SearchModeFullText`, `SearchModeTrigram`, dan
`SearchModeFullTextTrigram` hanya diterima untuk dialect PostgreSQL. Kolom
`FullTextColumn` harus bertipe `tsvector`; ORM tidak membuat atau menyinkronkan
kolom tersebut.

Contoh index—nama tabel/kolom harus disesuaikan dengan schema aplikasi:

```sql
CREATE INDEX idx_user_profile_search_fts_keyword
ON user_profile_search
USING GIN (fts_keyword);
```

Fallback `ILIKE '%keyword%'` dapat menggunakan GIN trigram index pada
PostgreSQL. Extension dan index harus dibuat oleh migration aplikasi:

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_user_profile_search_fts_lexeme_text_trgm
ON user_profile_search
USING GIN (fts_lexeme_text gin_trgm_ops);
```

Index trigram tidak mengubah fallback menjadi similarity atau fuzzy search; ia
hanya dapat mempercepat kondisi `ILIKE` yang sudah ada. PostgreSQL tetap dapat
memilih sequential scan, terutama untuk tabel kecil atau pola yang kurang
selektif.

Search document harus tetap sinkron dengan sumber seperti email, nama, alamat,
atau telepon. Pilih mekanisme yang sesuai schema: generated column bila
ekspresinya memenuhi batasan PostgreSQL, trigger database, atau update eksplisit
dalam proses aplikasi. Contoh index di atas bukan migration lengkap dan tidak
mengasumsikan struktur `user_profile_search` selain tipe kolom yang dijelaskan.

## Validasi dan Error

Library mengembalikan dictionary error berikut:

| Kondisi | Error aktual |
| --- | --- |
| Nilai `SearchMode` tidak dikenal | `dictionary.ErrInvalidSearchMode` |
| Mode tidak tercantum dalam `SearchFieldConfig.Modes` | `dictionary.ErrSearchModeNotAllowedForField` |
| Alias advanced search tidak ada di `SearchFields` | `dictionary.ErrSearchFieldNotAllowed` |
| `FullTextColumn` kosong untuk mode full-text | `dictionary.ErrFullTextColumnRequired` |
| `FullTextLanguage` bukan `simple`, `english`, atau kosong | `dictionary.ErrInvalidFullTextLanguage` |
| Mode full-text/trigram dipakai pada non-PostgreSQL | `dictionary.ErrUnsupportedSearchModeForDialect` |
| Field sort tidak ada di `AllowedFields` | `dictionary.ErrColumnNotFound` |
| `InMemoryOffset` tidak memiliki field cursor atau nil value | `dictionary.ErrPaginationCursorRequired` |

Pada `grpc-auth`, sort cursor selain `auth_id` menghasilkan validation error
consumer. `last_id` nonzero yang bukan integer positif juga menghasilkan error
terstruktur pada field `last_id` dengan pesan `cursor must be a positive
integer`; ini bukan dictionary error ORM. Cursor awal `"0"` valid.

## Catatan Performa

- Buat GIN index pada `tsvector` dan, jika fallback penting, GIN
  `gin_trgm_ops` pada kolom teks yang tepat.
- Kondisi `OR` dapat membuat planner menggabungkan index atau memilih scan;
  periksa `EXPLAIN (ANALYZE, BUFFERS)` dengan distribusi data produksi.
- `QueryPageWithConfig` menjalankan count query selain data query. Pada dataset
  besar, biaya count dan kondisi `OR` perlu diukur.
- Jalur cursor `grpc-auth` mengambil hingga 1000 row per batch lalu melakukan
  slice page di memori. Sesuaikan strategi consumer jika ukuran row besar atau
  pola akses membutuhkan keyset pagination murni.
- Implementasi tidak menghitung rank. Urutan hasil sepenuhnya berasal dari
  `Sort`; contoh memakai `a.id ASC`.

## Contoh Lengkap

Contoh berikut dapat dikompilasi setelah dependency ORM tersedia dan adapter
database PostgreSQL yang valid diberikan:

```go
package usersearch

import (
    "context"

    "github.com/siti-nabila/orm/orm"
)

type UserSearchRow struct {
    AuthID  int64  `sql:"column:auth_id" json:"auth_id"`
    Email   string `sql:"column:email" json:"email"`
    Name    string `sql:"column:name" json:"name"`
    Address string `sql:"column:address" json:"address"`
    Phone   string `sql:"column:phone" json:"phone"`
}

func (UserSearchRow) TableName() string { return "profile p" }

func Search(
    ctx context.Context,
    db *orm.SqlQueryAdapter,
) (orm.PageData[UserSearchRow], error) {
    opts := orm.QueryOptions{
        Page:  1,
        Limit: 2,
        Sort: []orm.SortField{
            {Field: "auth_id", Desc: false},
        },
        Search: &orm.SearchQuery{
            Fields:  []string{"keyword"},
            Keyword: "nabila",
            Mode:    orm.SearchModeFullTextTrigram,
        },
    }

    rows := make([]UserSearchRow, 0)
    query := db.UseModel(UserSearchRow{}).
        Select(
            "a.id AS auth_id",
            "a.email",
            `p."name"`,
            "p.address",
            "p.phone",
        ).
        Join("auth a", "a.id = p.user_id").
        Join("user_profile_search ups", "ups.profile_id = p.id")

    pageData, err := orm.QueryPageWithConfig(
        ctx,
        query,
        &rows,
        orm.QueryPageConfig{
            AllowedFields: map[string]string{
                "auth_id": "a.id",
                "email":   "a.email",
                "name":    `p."name"`,
                "phone":   "p.phone",
            },
            SearchFields: map[string]orm.SearchFieldConfig{
                "keyword": {
                    Column:           "ups.fts_lexeme_text",
                    FullTextColumn:   "ups.fts_keyword",
                    FullTextLanguage: orm.FullTextSimple,
                    Modes: []orm.SearchMode{
                        orm.SearchModeFullText,
                        orm.SearchModeFullTextTrigram,
                    },
                },
            },
        },
        opts,
    )
    if err != nil {
        return orm.PageData[UserSearchRow]{}, err
    }
    return pageData, nil
}
```
