# Paginasi

Repositori ini memiliki dua gaya paginasi:

- paginasi database melalui query builder
- paginasi slice dalam memori melalui `pagination.SlicePaginator`

Keduanya dapat menghasilkan respons frontend yang deterministik dengan `PageData[T]`.

## PageMeta

`ScanPaginate` mengembalikan metadata paginasi sebagai `PageMeta`:

```go
type PageMeta struct {
    Page       int
    Limit      int
    Total      int64
    TotalPages int
    HasNext    bool
    HasPrev    bool
}
```

`Page` dinormalisasi minimal menjadi `1`. `Limit` adalah ukuran halaman yang
dinormalisasi dari `PaginationOptions.PerPage`. `TotalPages` menggunakan
aritmetika integer. Ketika total baris `0`, `TotalPages` bernilai `0`, `HasNext`
bernilai `false`, dan `HasPrev` bernilai `false`.

## PageData

`PageData[T]` adalah bentuk respons akhir untuk paginasi yang digunakan frontend:

```go
type PageData[T any] struct {
    Items      []T    `json:"items"`
    Total      int64  `json:"total"`
    Page       int    `json:"page"`
    Limit      int    `json:"limit"`
    TotalPages int    `json:"total_pages"`
    HasNext    bool   `json:"has_next"`
    HasPrev    bool   `json:"has_prev"`
    NextCursor string `json:"next_cursor,omitempty"`
}
```

`Items` selalu berupa slice. Hasil kosong menggunakan `Items: []T{}`.

Contoh respons:

```json
{
  "items": [
    {
      "id": 1,
      "name": "Nabila",
      "email": "nabila@example.com",
      "status": "ACTIVE",
      "joinDate": "2026-01-10"
    }
  ],
  "total": 55,
  "page": 1,
  "limit": 20,
  "total_pages": 3,
  "has_next": true,
  "has_prev": false,
  "next_cursor": "1"
}
```

Respons kosong:

```json
{
  "items": [],
  "total": 0,
  "page": 1,
  "limit": 20,
  "total_pages": 0,
  "has_next": false,
  "has_prev": false
}
```

Ketika terjadi error dalam alur helper `PageData[T]`, fungsi mengembalikan
`PageData[T]` kosong dan error secara terpisah. Error tidak disimpan dalam
`PageData[T]`.

## ScanPaginate

`ScanPaginate` adalah paginator query inti. Method ini menjalankan query count,
lalu query data dengan paginasi:

```go
var users []User

builder, err := db.
    UseModel(User{}).
    WhereOp("status", orm.OpEqual, "ACTIVE")
if err != nil {
    return err
}

pageMeta, err := builder.
    OrderBy("created_at DESC").
    ScanPaginate(ctx, &users, orm.PaginationOptions{
        Page:    1,
        PerPage: 20,
    })
if err != nil {
    return err
}

fmt.Println(pageMeta.Total)
fmt.Println(pageMeta.TotalPages)
```

Query count menghapus pengurutan dan paginasi. Bentuk query sederhana menggunakan:

```sql
SELECT COUNT(*) FROM users WHERE ...
```

Bentuk query kompleks menggunakan count terbungkus:

```sql
SELECT COUNT(*) FROM (SELECT ...) count_table
```

Count terbungkus digunakan untuk `DISTINCT`, `GROUP BY`, `HAVING`, ekspresi
select mentah, dan ekspresi select alias. Query data mempertahankan bentuk query
yang dipilih dan menerapkan `LIMIT/OFFSET` atau klausa paginasi Oracle.
`ScanPaginate` tidak mengubah perilaku `Scan`, `First`, `Limit`, atau `Offset`
yang sudah ada.

`Where` tetap didukung. `WhereOp` lebih disarankan untuk operator bertipe karena
memetakan konstanta `orm.Operator` seperti `orm.OpEqual` dan
`orm.OpGreaterThanEqual` ke SQL secara internal.

### Offset dalam Memori

Secara default, `ScanPaginate` menggunakan paginasi offset database:

```sql
LIMIT n OFFSET m
```

Gunakan `InMemoryOffset` ketika pemanggil ingin query database membaca batch
cursor terbatas, lalu menerapkan slicing `Page`/`PerPage` dalam Go:

```go
var users []User

pageMeta, err := db.UseModel(User{}).
    WhereOp("id", orm.OpGreaterThan, lastID).
    OrderBy("id ASC").
    ScanPaginate(ctx, &users, orm.PaginationOptions{
        Page:    2,
        PerPage: 20,
        InMemoryOffset: &orm.PaginationInMemoryOffsetOptions{
            CursorField: "id",
            MaxLimit: 1000,
        },
    })
```

Dalam mode `ScanPaginate` tingkat rendah ini, pemanggil harus menambahkan kondisi
cursor secara eksplisit, seperti `id > lastID` untuk urutan ascending atau
`id < lastID` untuk urutan descending. Atur `CursorField` ketika Anda ingin
paginator mengembalikan `PageMeta.NextCursor`. Query data menggunakan
`LIMIT MaxLimit` dan tidak menghasilkan `OFFSET`.

Contoh bentuk query data PostgreSQL:

```sql
SELECT ... FROM users
WHERE id > $1
ORDER BY id ASC
LIMIT 1000
```

Setelah memindai batch, ORM melakukan slicing hasil dalam memori menggunakan
`Page`/`PerPage`. `NextCursor` diambil dari baris terakhir pada halaman yang
ditampilkan setelah slicing dalam memori, sehingga client dapat mengirimkannya
sebagai cursor untuk request berikutnya.

Karena `Page` tetap merepresentasikan slice yang ditampilkan di dalam batch
cursor saat ini, `HasPrev` mengikuti metadata halaman yang diminta. Misalnya,
halaman `2` dalam batch cursor 1000 baris memiliki `has_prev: true`. Kembali ke
halaman `1` tidak memerlukan cache di sisi ORM; pemanggil dapat meminta halaman
`1` lagi dengan input cursor yang sama dan ORM akan meng-query ulang batch,
kemudian mengambil slice halaman pertama.

Untuk `QueryPageWithConfig`, gunakan `QueryOptions.InMemoryOffset`. Opsi ini
menempatkan cursor di bawah mode dalam memori, sehingga paginasi query normal
tidak memerlukan field cursor:

```go
opts := orm.QueryOptions{
    Page:  2,
    Limit: 20,
    InMemoryOffset: &orm.InMemoryOffsetOptions{
        Cursor: orm.Cursor{
            Field: "id",
            Value: lastID,
        },
        MaxLimit: 1000,
    },
    Sort: []orm.SortField{{Field: "id"}},
}
```

`Cursor.Field` di-resolve melalui `AllowedFields`. Untuk urutan ascending, ORM
menambahkan `> cursor`, dan untuk descending menambahkan `< cursor`. Untuk batch
pertama, atur `Cursor.Value` menjadi string kosong; ORM mempertahankan mode
offset dalam memori tetapi tidak menyertakan predikat cursor. Jika
`InMemoryOffset` diatur dengan `Cursor.Field` kosong atau `Cursor.Value` nil,
`QueryPageWithConfig` mengembalikan dictionary error. `PageData.NextCursor`
diisi dari field cursor yang sama setelah slicing dalam memori dan dienkode
sebagai `next_cursor` dalam respons JSON.

### Pencarian Full Text Profil PostgreSQL

Untuk kolom pencarian `tsvector` PostgreSQL, gunakan `WhereFullText` sebagai
pengganti operator perbandingan generik. Helper menghasilkan:

```sql
<column> @@ websearch_to_tsquery('simple', <arg>)
```

Contoh dengan data profile/auth dan tabel pencarian yang dipelihara trigger:

```go
type ProfileSearchRow struct {
    AuthID    int64  `sql:"column:auth_id"`
    Email     string `sql:"column:email"`
    ProfileID int64  `sql:"column:profile_id"`
    Name      string `sql:"column:name"`
    Address   string `sql:"column:address"`
    Phone     string `sql:"column:phone"`
}

func (ProfileSearchRow) TableName() string {
    return "profile p"
}

var rows []ProfileSearchRow

builder, err := db.UseModel(ProfileSearchRow{}).
    Select(
        "a.id AS auth_id",
        "a.email",
        "p.id AS profile_id",
        "p.\"name\"",
        "p.address",
        "p.phone",
    ).
    Join("auth a", "a.id = p.user_id").
    Join("user_profile_search ups", "ups.profile_id = p.id").
    WhereFullText("ups.fts_keyword", keyword)
if err != nil {
    return err
}

pageMeta, err := builder.
    OrderBy("p.id DESC").
    ScanPaginate(ctx, &rows, orm.PaginationOptions{
        Page:    1,
        PerPage: 20,
    })
```

`user_profile_search.fts_keyword` adalah kolom `tsvector` PostgreSQL yang
dipelihara oleh trigger database, misalnya dari `auth.email`, `profile."name"`,
dan `profile.phone`. Query count paginator menggunakan join dan kondisi full text
yang sama dengan query data, sehingga `TotalRows` sesuai dengan hasil filter.

Tambahkan index GIN untuk kolom pencarian:

```sql
CREATE INDEX idx_user_profile_search_fts_keyword
ON user_profile_search
USING GIN (fts_keyword);
```

Untuk `QueryPageWithConfig`, `SearchModeFullTextTrigram` dapat menggabungkan
pencarian prefix full text PostgreSQL dengan pencarian contains pada kolom teks
yang dihasilkan dari lexeme `tsvector`. Dalam mode ini:

- `FullTextColumn` adalah kolom `tsvector`, misalnya `ups.fts_keyword`.
- `Column` adalah kolom teks lexeme, misalnya `ups.fts_lexeme_text`.

```go
pageData, err := orm.QueryPageWithConfig(
    ctx,
    db.UseModel(ProfileSearchRow{}).
        Select(
            "a.id AS auth_id",
            "a.email",
            "p.id AS profile_id",
            "p.\"name\"",
            "p.address",
            "p.phone",
        ).
        Join("auth a", "a.id = p.user_id").
        Join("user_profile_search ups", "ups.profile_id = p.id"),
    &rows,
    orm.QueryPageConfig{
        AllowedFields: map[string]string{
            "profileID": "p.id",
        },
        SearchFields: map[string]orm.SearchFieldConfig{
            "keyword": {
                Column:         "ups.fts_lexeme_text",
                FullTextColumn: "ups.fts_keyword",
                Modes:          []orm.SearchMode{orm.SearchModeFullTextTrigram},
            },
        },
    },
    orm.QueryOptions{
        Page:  1,
        Limit: 20,
        Search: &orm.SearchQuery{
            Fields:  []string{"keyword"},
            Keyword: "siti nab",
            Mode:    orm.SearchModeFullTextTrigram,
        },
        Sort: []orm.SortField{{Field: "profileID", Desc: true}},
    },
)
```

Kondisi PostgreSQL yang dihasilkan setara dengan:

```sql
ups.fts_keyword @@ to_tsquery('simple', $1)
OR ups.fts_lexeme_text ILIKE '%' || lower($2) || '%'
```

ORM mengonversi keyword pengguna biasa menjadi argumen yang aman:

```go
[]any{"siti:* & nab:*", "siti nab"}
```

`user_profile_search.fts_lexeme_text` adalah kolom teks yang dihasilkan dari
lexeme `fts_keyword`, misalnya dengan
`array_to_string(tsvector_to_array(fts_keyword), ' ')`.

Index yang disarankan:

```sql
CREATE INDEX idx_user_profile_search_fts_keyword
ON user_profile_search
USING GIN (fts_keyword);

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_user_profile_search_fts_lexeme_text_trgm
ON user_profile_search
USING GIN (fts_lexeme_text gin_trgm_ops);
```

## QueryPage

`QueryPage` adalah helper tingkat tinggi untuk paginasi yang dikendalikan
frontend. Pengguna tetap membuat query yang terikat model secara eksplisit
dengan `UseModel`.

Gunakan `ScanPaginate` ketika Anda menginginkan alur inti ORM serta items dan
`PageMeta` secara terpisah. Gunakan `QueryPage` ketika Anda ingin opsi frontend
dipetakan melalui field yang diizinkan dan dikembalikan sebagai satu nilai
`PageData[T]`.

```go
type UserReader struct {
    ctx context.Context
    db  *orm.SqlQueryAdapter
}

// AllowedFields mengembalikan pemetaan field frontend-ke-database yang diizinkan.
//
// Key map adalah nama field JSON/request yang digunakan frontend.
// Value map adalah nama kolom database aktual yang digunakan query builder.
func (r *UserReader) AllowedFields() map[string]string {
    return map[string]string{
        // field frontend -> kolom database
        "id":       "id",
        "name":     "name",
        "email":    "email",
        "status":   "status",
        "joinDate": "join_date",
    }
}

func (r *UserReader) Page(opts orm.QueryOptions) (orm.PageData[User], error) {
    var items []User

    q := r.db.UseModel(User{})

    return orm.QueryPage(
        r.ctx,
        q,
        &items,
        r.AllowedFields(),
        opts,
    )
}
```

`AllowedFields()` memetakan nama field frontend ke kolom database. `QueryPage`
menggunakan pemetaan ini untuk `Select`, `Filters`, `Search`, `SearchAnd`, dan
`Sort`. Nama field frontend tidak diteruskan langsung ke query builder. Field
yang tidak terdapat dalam map ditolak, dan SQL mentah dari input frontend tidak
boleh diterima.

```go
type QueryOptions struct {
    Page           int
    Limit          int
    InMemoryOffset *InMemoryOffsetOptions
    Sort           []SortField
    Search         *SearchQuery
    SearchAnd      *SearchQueryAnd
    Filters        []Filter
    Select         []string
}

type SortField struct {
    Field string
    Desc  bool
}

type Filter struct {
    Field    string
    Operator Operator
    Value    string
    Values   []string
}

type SearchQuery struct {
    Fields  []string
    Keyword string
    Mode    SearchMode
}

type SearchField struct {
    Field   string
    Keyword string
}

type SearchQueryAnd struct {
    Fields []*SearchField
}
```

`QueryOptions.Limit` dipetakan ke `PaginationOptions.PerPage`, dan
`QueryOptions.Page` dipetakan ke `PaginationOptions.Page`.

Contoh opsi:

```go
opts := orm.QueryOptions{
    Page:  1,
    Limit: 20,
    Select: []string{"id", "name", "email", "joinDate"},
    Filters: []orm.Filter{
        {Field: "status", Operator: orm.OpEqual, Value: "ACTIVE"},
        {Field: "joinDate", Operator: orm.OpGreaterThanEqual, Value: "2026-01-01"},
        {Field: "joinDate", Operator: orm.OpLessThan, Value: "2026-03-23"},
        {Field: "status", Values: []string{"ACTIVE", "PENDING"}},
    },
    Search: &orm.SearchQuery{
        Fields:  []string{"name", "email"},
        Keyword: "nabila",
    },
    SearchAnd: &orm.SearchQueryAnd{
        Fields: []*orm.SearchField{
            {Field: "status", Keyword: "ACTIVE"},
        },
    },
    Sort: []orm.SortField{
        {Field: "joinDate", Desc: true},
    },
}
```

Filter dengan `Values` menggunakan `WhereIn`. Filter tanpa `Values` menggunakan
`WhereOp`. `Search` menerapkan predikat bergaya OR pada field yang diizinkan.
`SearchAnd` menerapkan predikat contains bergaya AND per field yang diizinkan.
Arah pengurutan ditentukan dari `Desc`.

## Mode Pencarian

`SearchQuery.Mode` mengendalikan cara `Search` membangun kondisi keyword:

```go
const (
    SearchModeContains        SearchMode = "contains"
    SearchModePrefix          SearchMode = "prefix"
    SearchModeFullText        SearchMode = "full_text"
    SearchModeTrigram         SearchMode = "trigram"
    SearchModeFullTextTrigram SearchMode = "full_text_trigram"
)
```

Jika `Mode` kosong, `QueryPage` menggunakan `SearchModeContains`. Pencarian
contains bersifat portabel di PostgreSQL, MySQL, dan Oracle:

```sql
column LIKE ?
```

dengan argumen seperti:

```go
"%keyword%"
```

`SearchModeContains` praktis digunakan, tetapi `LIKE '%keyword%'` mungkin tidak
ramah index pada tabel besar. `SearchModePrefix` juga portabel dan menggunakan
argumen seperti `keyword%`, yang dapat lebih ramah index ketika database dan
index mendukung pencocokan prefix.

Mode yang dioptimalkan hanya tersedia untuk PostgreSQL pada tahap ini:

- `SearchModeFullText` menggunakan kolom `tsvector` yang dikonfigurasi dengan `websearch_to_tsquery`.
- `SearchModeTrigram` menggunakan `ILIKE ?` pada kolom sumber yang dikonfigurasi dan mengasumsikan pemilik aplikasi dapat membuat index `pg_trgm`.
- `SearchModeFullTextTrigram` menggunakan kolom `tsvector` untuk pencarian prefix full text dan kolom teks lexeme untuk pencarian contains.

MySQL dan Oracle hanya mendukung `contains` dan `prefix` pada tahap ini. Meminta
`full_text`, `trigram`, atau `full_text_trigram` pada MySQL atau Oracle akan
mengembalikan dictionary error.

Mode pencarian lanjutan menggunakan `QueryPageWithConfig`:

```go
type QueryPageConfig struct {
    // AllowedFields memetakan field JSON/request frontend ke kolom database.
    // Key map adalah nama field JSON/request yang digunakan frontend.
    // Value map adalah nama kolom database aktual yang digunakan query builder.
    AllowedFields map[string]string
    SearchFields  map[string]SearchFieldConfig
}

type SearchFieldConfig struct {
    Column           string
    FullTextColumn   string
    FullTextLanguage FullTextLanguage
    FullTextOperator FullTextOperator
    Modes            []SearchMode
}
```

Contoh:

```go
func (r *UserReader) SearchFields() map[string]orm.SearchFieldConfig {
    return map[string]orm.SearchFieldConfig{
        "name": {
            Column:         "name",
            FullTextColumn: "fts_keyword",
            Modes: []orm.SearchMode{
                orm.SearchModeFullText,
                orm.SearchModeTrigram,
                orm.SearchModeFullTextTrigram,
            },
        },
    }
}

func (r *UserReader) Page(opts orm.QueryOptions) (orm.PageData[User], error) {
    var items []User

    return orm.QueryPageWithConfig(
        r.ctx,
        r.db.UseModel(User{}),
        &items,
        orm.QueryPageConfig{
            AllowedFields: r.AllowedFields(),
            SearchFields:  r.SearchFields(),
        },
        opts,
    )
}
```

`FullTextLanguage` adalah enum, bukan string mentah dari frontend:

```go
const (
    FullTextSimple  FullTextLanguage = "simple"
    FullTextEnglish FullTextLanguage = "english"
)
```

Bahasa kosong menggunakan default `FullTextSimple`. Bahasa yang tidak valid
mengembalikan dictionary error.

`FullTextOperator` terpisah dari `Operator` normal yang digunakan oleh filter dan
`WhereOp`:

```go
const (
    FullTextMatch       FullTextOperator = "match"
    FullTextContains    FullTextOperator = "contains"
    FullTextContainedBy FullTextOperator = "contained_by"
)
```

`FullTextOperator` hanya digunakan untuk mode pencarian full text. Jangan
menambahkan `@@`, `<@`, atau `@>` pada penggunaan normal `WhereOp`. Operator
kosong menggunakan default `FullTextMatch`.

Untuk `SearchModeFullTextTrigram`, keyword dipisahkan berdasarkan whitespace.
Setiap token menerima suffix prefix PostgreSQL untuk argumen full text. Dengan:

```text
joko yono wo
```

argumen prefix full text menjadi:

```go
"joko:* & yono:* & wo:*"
```

dan argumen contains menjadi:

```go
"joko yono wo"
```

Kondisi yang dihasilkan menggunakan `to_tsquery` untuk bagian prefix dan `ILIKE`
untuk bagian contains teks lexeme. Ranking dengan `ts_rank` atau `ts_rank_cd`,
pengurutan similarity, dan field ranking/similarity publik berada di luar cakupan
tahap ini.

ORM tidak membuat extension atau index PostgreSQL. Pemilik aplikasi harus
menambahkan migration saat menggunakan pencarian yang dioptimalkan.

Contoh index full text:

```sql
CREATE INDEX users_fts_keyword_idx
ON users
USING GIN (fts_keyword);
```

Contoh konfigurasi trigram:

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX users_name_trgm_idx
ON users
USING GIN (name gin_trgm_ops);
```

## SlicePaginator

`pagination.SlicePaginator` melakukan paginasi data yang sudah dimuat dalam
memori dan mengembalikan `pagination.PageData[T]`.

```go
pageData, err := pagination.
    FromSlice(users).
    Filter(func(user User) bool {
        return user.Status == "ACTIVE"
    }).
    Sort(func(a, b User) bool {
        return a.CreatedAt > b.CreatedAt
    }).
    Paginate(pagination.Params{
        Page:  1,
        Limit: 20,
    })
if err != nil {
    return err
}

fmt.Println(pageData.Items)
fmt.Println(pageData.Total)
```

Tipe dan fungsi paginasi slice publik:

- `SlicePaginator`
- `FromSlice`
- `FromSliceWithConfig`
- `Filter`
- `Sort`
- `Paginate`
- `Params`
- `PageData[T]`
- `Config`

`FromSlice` menyalin slice input. `Filter` menambahkan predikat non-nil. `Sort`
menggunakan `sort.SliceStable`. `Paginate` menormalisasi parameter, menerapkan
semua filter, menerapkan stable sort, menghitung total setelah filter, dan
mengembalikan `PageData[T]` dengan `Items` non-nil.
