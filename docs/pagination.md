# Pagination

The repository has two pagination styles:

- database pagination through query builders
- in-memory slice pagination through `pagination.SlicePaginator`

Both can produce deterministic frontend responses with `PageData[T]`.

## PageMeta

`ScanPaginate` returns pagination metadata as `PageMeta`:

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

`Page` is normalized to at least `1`. `Limit` is the normalized page size from
`PaginationOptions.PerPage`. `TotalPages` uses integer arithmetic. When total
rows is `0`, `TotalPages` is `0`, `HasNext` is `false`, and `HasPrev` is
`false`.

## PageData

`PageData[T]` is the final response shape for frontend-facing pagination:

```go
type PageData[T any] struct {
    Items      []T   `json:"items"`
    Total      int64 `json:"total"`
    Page       int   `json:"page"`
    Limit      int   `json:"limit"`
    TotalPages int   `json:"total_pages"`
    HasNext    bool  `json:"has_next"`
    HasPrev    bool  `json:"has_prev"`
}
```

`Items` is always a slice. Empty results use `Items: []T{}`.

Example response:

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
  "has_prev": false
}
```

Empty response:

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

When an error occurs in a `PageData[T]` helper flow, the function returns an
empty `PageData[T]` and returns the error separately. Errors are not stored in
`PageData[T]`.

## ScanPaginate

`ScanPaginate` is the core query paginator. It runs a count query, then a data
query with pagination:

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

The count query removes ordering and pagination. Simple query shapes use:

```sql
SELECT COUNT(*) FROM users WHERE ...
```

Complex query shapes use a wrapped count:

```sql
SELECT COUNT(*) FROM (SELECT ...) count_table
```

Wrapped count is used for `DISTINCT`, `GROUP BY`, `HAVING`, raw select
expressions, and alias select expressions. The data query keeps the selected
query shape and applies `LIMIT/OFFSET` or the Oracle pagination clause.
`ScanPaginate` does not change existing `Scan`, `First`, `Limit`, or `Offset`
behavior.

`Where` is still supported. `WhereOp` is preferred for typed operators because
it maps `orm.Operator` constants such as `orm.OpEqual` and
`orm.OpGreaterThanEqual` to SQL internally.

## QueryPage

`QueryPage` is a high-level helper for frontend-driven pagination. Users still
create the model-bound query explicitly with `UseModel`.

Use `ScanPaginate` when you want the core ORM flow and separate items plus
`PageMeta`. Use `QueryPage` when you want frontend options mapped through
allowed fields and returned as one `PageData[T]` value.

```go
type UserReader struct {
    ctx context.Context
    db  *orm.SqlQueryAdapter
}

// AllowedFields returns the allowed frontend-to-database field mapping.
//
// The map key is the JSON/request field name used by the frontend.
// The map value is the actual database column name used by the query builder.
func (r *UserReader) AllowedFields() map[string]string {
    return map[string]string{
        // frontend field -> database column
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

`AllowedFields()` maps frontend field names to database columns. `QueryPage`
uses this mapping for `Select`, `Filters`, `Search`, `SearchAnd`, and `Sort`.
Frontend field names are not passed directly to the query builder. Fields that
are not present in the map are rejected, and raw SQL from frontend input should
not be accepted.

```go
type QueryOptions struct {
    Page      int
    Limit     int
    Sort      []SortField
    Search    *SearchQuery
    SearchAnd *SearchQueryAnd
    Filters   []Filter
    Select    []string
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

`QueryOptions.Limit` maps to `PaginationOptions.PerPage`, and
`QueryOptions.Page` maps to `PaginationOptions.Page`.

Example options:

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

Filters with `Values` use `WhereIn`. Filters without `Values` use `WhereOp`.
`Search` applies OR-style predicates across allowed fields. `SearchAnd` applies
AND-style contains predicates per allowed field. Sorting direction is derived
from `Desc`.

## Search Modes

`SearchQuery.Mode` controls how `Search` builds keyword conditions:

```go
const (
    SearchModeContains        SearchMode = "contains"
    SearchModePrefix          SearchMode = "prefix"
    SearchModeFullText        SearchMode = "full_text"
    SearchModeTrigram         SearchMode = "trigram"
    SearchModeFullTextTrigram SearchMode = "full_text_trigram"
)
```

If `Mode` is empty, `QueryPage` uses `SearchModeContains`. Contains search is
portable across PostgreSQL, MySQL, and Oracle:

```sql
column LIKE ?
```

with an argument like:

```go
"%keyword%"
```

`SearchModeContains` is convenient, but `LIKE '%keyword%'` may not be
index-friendly on large tables. `SearchModePrefix` is also portable and uses an
argument like `keyword%`, which can be more index-friendly when the database
and index support prefix matching.

The optimized modes are PostgreSQL-only in this phase:

- `SearchModeFullText` uses a configured `tsvector` column.
- `SearchModeTrigram` uses `ILIKE ?` on the configured source column and assumes the application owner may create a `pg_trgm` index.
- `SearchModeFullTextTrigram` uses full-text search for completed tokens and `ILIKE ?` for the final partial token.

MySQL and Oracle support only `contains` and `prefix` in this phase. Asking for
`full_text`, `trigram`, or `full_text_trigram` on MySQL or Oracle returns a
dictionary error.

Advanced search modes use `QueryPageWithConfig`:

```go
type QueryPageConfig struct {
    // AllowedFields maps frontend JSON/request fields to database columns.
    // The map key is the JSON/request field name used by the frontend.
    // The map value is the actual database column name used by the query builder.
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

Example:

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

`FullTextLanguage` is an enum, not a raw frontend string:

```go
const (
    FullTextSimple  FullTextLanguage = "simple"
    FullTextEnglish FullTextLanguage = "english"
)
```

Empty language defaults to `FullTextSimple`. Invalid languages return a
dictionary error.

`FullTextOperator` is separate from the normal `Operator` used by filters and
`WhereOp`:

```go
const (
    FullTextMatch       FullTextOperator = "match"
    FullTextContains    FullTextOperator = "contains"
    FullTextContainedBy FullTextOperator = "contained_by"
)
```

`FullTextOperator` is only for full-text search modes. Do not add `@@`, `<@`,
or `@>` to normal `WhereOp` usage. Empty operator defaults to
`FullTextMatch`.

For `SearchModeFullTextTrigram`, the keyword is split by whitespace. With:

```text
joko yono wo
```

the completed full-text tokens become:

```go
"joko & yono"
```

and the partial token becomes:

```go
"%wo%"
```

If the keyword has one token, the hybrid mode uses full-text search only.
Ranking with `ts_rank` or `ts_rank_cd`, similarity ordering, and public
ranking/similarity fields are out of scope for this phase.

The ORM does not create PostgreSQL extensions or indexes. Application owners
should add migrations when using optimized search.

Full-text index example:

```sql
CREATE INDEX users_fts_keyword_idx
ON users
USING GIN (fts_keyword);
```

Trigram setup example:

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX users_name_trgm_idx
ON users
USING GIN (name gin_trgm_ops);
```

## SlicePaginator

`pagination.SlicePaginator` paginates data already loaded in memory and returns
`pagination.PageData[T]`.

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

Public slice pagination types and functions:

- `SlicePaginator`
- `FromSlice`
- `FromSliceWithConfig`
- `Filter`
- `Sort`
- `Paginate`
- `Params`
- `PageData[T]`
- `Config`

`FromSlice` copies the input slice. `Filter` appends non-nil predicates.
`Sort` uses `sort.SliceStable`. `Paginate` normalizes params, applies all
filters, applies stable sort, calculates totals after filtering, and returns
`PageData[T]` with non-nil `Items`.
