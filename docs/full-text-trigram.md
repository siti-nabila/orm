# PostgreSQL Full Text Trigram Search

## Overview

`orm.SearchModeFullTextTrigram` combines two PostgreSQL conditions with `OR`:

- full-text prefix search on a `tsvector` column through `to_tsquery`;
- a case-insensitive contains fallback on a configured text column through
  `ILIKE`.

The mode name refers to the ability to optimize the `ILIKE` fallback with a
`pg_trgm` index. The ORM does not use the similarity operator (`%`), the
`similarity` function, or typo matching. Fallback results are still determined
by `ILIKE '%keyword%'`.

This guide follows the ORM API used by `grpc-auth` at
`github.com/siti-nabila/orm v1.7.0` and the current consumer implementation.

The actual search modes differ as follows:

| Mode | Condition | Dialect | Notes |
| --- | --- | --- | --- |
| `SearchModeContains` | `Column LIKE ?`, argument `%keyword%` | PostgreSQL, MySQL, Oracle | Default when `Mode` is empty; case sensitivity follows the database/collation. |
| `SearchModePrefix` | `Column LIKE ?`, argument `keyword%` | PostgreSQL, MySQL, Oracle | Plain text prefix, not full-text search. |
| `SearchModeFullText` | `FullTextColumn <operator> websearch_to_tsquery('<language>', ?)` | PostgreSQL | The default operator is `@@`; the keyword is passed unchanged as an argument. |
| `SearchModeTrigram` | `Column ILIKE ?`, argument `%keyword%` | PostgreSQL | It does not calculate similarity and is not fuzzy; the name indicates that `pg_trgm` can index this pattern. |
| `SearchModeFullTextTrigram` | `FullTextColumn @@ to_tsquery(...) OR Column ILIKE ...` | PostgreSQL | Full-text prefix plus contains fallback; the current full-text branch always uses `@@`. |

## When to Use Full Text Trigram

Use this mode when an application needs per-token prefixes—for example, `nab`
matching a lexeme that starts with `nab`—and a substring fallback against a text
search document. It is suitable when:

- the full-text search document is already stored as `tsvector`;
- the consumer can provide a text fallback column representing the searchable
  source data;
- PostgreSQL is the active database;
- typo matching and similarity ranking are not required. This implementation
  does not provide either feature.

## Request

Main `grpc-auth` `ListUsers` request example:

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

- `page` is a one-based page number. `grpc-auth` preserves it as the in-batch
  offset page for cursor pagination.
- `limit` is the displayed item count. A value of zero or less is normalized by
  the consumer to the default 10; a value above the maximum is capped at
  `pagination.MaxLimit` (1000 in this version).
- `sort` controls result order. `field: "auth_id"` maps to `a.id`, while
  `desc: false` produces `ASC`. The `grpc-auth` user cursor only accepts an
  `auth_id` sort.
- `search.fields` selects public search aliases. `keyword` is a key in
  `SearchFields`; it is **not** a database column name.
- `search.keyword` is the user's search text.
- `search.mode` selects the protobuf enum mapped to an ORM mode.
- `last_id` is the batch cursor. The string `"0"` (or an empty string) denotes
  the initial batch; a nonzero value must be a positive integer.

## Mapping the Request to ORM

The `grpc-auth` handler maps the enum explicitly:

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

The snippet uses these actual imports:

```go
import (
    "github.com/siti-nabila/grpc-auth/pb/paginator"
    "github.com/siti-nabila/orm/orm"
    ormdictionary "github.com/siti-nabila/orm/pkg/dictionary"
)
```

After the handler maps `PageQuery`, the main request produces:

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

`last_id` is not a `QueryOptions` field at the handler stage. The `grpc-auth`
service calls `paginator.Build` in `ModeCursor`, removes and reinstalls the
validated sort as a default sort, then adds these options before calling the
repository:

```go
opts.InMemoryOffset = &orm.InMemoryOffsetOptions{
    Cursor: orm.Cursor{
        Field: "auth_id",
        Value: "", // last_id "0" is normalized to an empty initial cursor
    },
    MaxLimit: pagination.MaxLimit,
}
```

For `last_id: "18"`, the consumer parser produces `Value: int64(18)`.

## Consumer Configuration

`grpc-auth` separates the general field allowlist from its search alias
configuration:

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

- `Column` is the text column expression used by the fallback. In this mode the
  ORM builds `Column ILIKE '%' || lower($n) || '%'`.
- `FullTextColumn` is the `tsvector` column expression used by the full-text
  condition.
- `FullTextLanguage` selects the text search configuration. Available values
  are `orm.FullTextSimple` and `orm.FullTextEnglish`; an empty value defaults to
  `simple`.
- `Modes` is the alias-specific mode allowlist. An empty list allows only the
  portable modes (`contains` and `prefix`), so advanced modes must be listed
  explicitly.

Request keys are mapped by the application to predefined database expressions.
Clients therefore cannot submit column names, join aliases, or SQL expressions
directly. `AllowedFields` is used for operations including sorting, while
`SearchFields` permits search aliases such as `keyword`.

## Running the Query

The consumer's actual result model is:

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

The consumer's base query is:

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

A complete ORM call:

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

Required imports:

```go
import (
    "context"

    "github.com/siti-nabila/orm/orm"
)
```

`QueryPageWithConfig` accepts a context, a model-bound query builder, a pointer
to the destination slice, mappings/allowlists, and `QueryOptions`. It runs a
count query and a data query, fills the slice, and returns `orm.PageData[T]`.
`PageData` contains `Items`, `Total`, `Page`, `Limit`, `TotalPages`, `HasNext`,
`HasPrev`, and `NextCursor`. Validation, SQL construction, database execution,
and scan errors are returned to the consumer, which can map them to gRPC/API
statuses.

## How Search Works

The keyword is split with `strings.Fields`. Every token receives a `:*` suffix
and is joined with ` & ` for the full-text argument. Fallback whitespace is
normalized to a single space:

```text
nabila       -> nabila:*
siti nabila  -> siti:* & nabila:*
```

For the main request, the conceptual parameterized PostgreSQL condition is:

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

This is the conceptual form for direct `QueryOptions` with a page size of 2.
The dialect/query builder determines final placeholders and query details; do
not interpolate keywords into SQL.

On the `grpc-auth` path, the service always adds `InMemoryOffset` with
`MaxLimit: pagination.MaxLimit`. The initial batch data query therefore
actually ends in `LIMIT 1000` without `OFFSET`, after which the ORM takes the
two page-1 items in memory. A separate count query uses the same search
conditions but has no `ORDER BY`, `LIMIT`, or `OFFSET`.

## Example Result

Illustrative data:

| auth_id | email | name |
| ---: | --- | --- |
| 12 | nabila@example.com | Siti Nabila |
| 18 | nabilah@example.com | Nabilah Putri |
| 25 | budi@example.com | Budi Santoso |

Example consumer response:

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

The data and all metadata above are illustrative only. The ORM derives
`NextCursor` from the cursor field on the last displayed item. The cursor can
therefore be present even when `HasNext` is `false`; use `HasNext` to determine
whether the total/page metadata indicates more results.

## Cursor Pagination

`"last_id": "0"` denotes the initial batch. `grpc-auth` converts it to an empty
cursor, so the ORM does not add an `a.id > ...` or `a.id < ...` condition.

If a response contains `"next_cursor": "18"`, the requested next-batch example
is:

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

The actual behavior matters: `grpc-auth` passes page 2 and cursor 18 together.
For ascending order, the ORM adds `a.id > $n`, fetches up to 1000 rows without
a SQL `OFFSET`, and then applies the page-2 offset (`(2-1)*2`) in memory. The
first two rows after ID 18 are consequently skipped. To request the first page
immediately after cursor 18, a consumer must send `page: 1`; this implementation
must not be treated as pure keyset pagination. Descending order uses `<`.

The count query uses the same cursor condition. For a request with a nonzero
`last_id`, `Total`, `TotalPages`, `HasNext`, and `HasPrev` are therefore computed
against the cursor-filtered set, not necessarily the complete initial search
result.

`NextCursor` comes from the `column:auth_id` SQL tag on the last item. A nonzero
`last_id` is validated by `grpc-auth` as a positive integer before the repository
is called.

## PostgreSQL and Index Requirements

`SearchModeFullText`, `SearchModeTrigram`, and
`SearchModeFullTextTrigram` are accepted only for the PostgreSQL dialect. The
`FullTextColumn` must have type `tsvector`; the ORM neither creates nor
synchronizes it.

Example index—the table and column names must be adapted to the application
schema:

```sql
CREATE INDEX idx_user_profile_search_fts_keyword
ON user_profile_search
USING GIN (fts_keyword);
```

The `ILIKE '%keyword%'` fallback can use a PostgreSQL GIN trigram index. The
application migration must create the extension and index:

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_user_profile_search_fts_lexeme_text_trgm
ON user_profile_search
USING GIN (fts_lexeme_text gin_trgm_ops);
```

A trigram index does not turn the fallback into similarity or fuzzy search; it
can only accelerate the existing `ILIKE` condition. PostgreSQL may still choose
a sequential scan, especially for small tables or poorly selective patterns.

The search document must remain synchronized with sources such as email, name,
address, or phone. Choose a schema-appropriate mechanism: a generated column
when the expression satisfies PostgreSQL restrictions, a database trigger, or
explicit updates in the application process. The indexes above are examples,
not a complete migration, and assume no `user_profile_search` structure beyond
the documented column types.

## Validation and Errors

The library returns these dictionary errors:

| Condition | Actual error |
| --- | --- |
| Unknown `SearchMode` value | `dictionary.ErrInvalidSearchMode` |
| Mode absent from `SearchFieldConfig.Modes` | `dictionary.ErrSearchModeNotAllowedForField` |
| Advanced search alias absent from `SearchFields` | `dictionary.ErrSearchFieldNotAllowed` |
| Empty `FullTextColumn` for a full-text mode | `dictionary.ErrFullTextColumnRequired` |
| `FullTextLanguage` other than `simple`, `english`, or empty | `dictionary.ErrInvalidFullTextLanguage` |
| Full-text/trigram mode on a non-PostgreSQL dialect | `dictionary.ErrUnsupportedSearchModeForDialect` |
| Sort field absent from `AllowedFields` | `dictionary.ErrColumnNotFound` |
| `InMemoryOffset` has an empty cursor field or nil value | `dictionary.ErrPaginationCursorRequired` |

In `grpc-auth`, a cursor sort other than `auth_id` produces a consumer validation
error. A nonzero `last_id` that is not a positive integer also produces a
structured error on `last_id` with the message `cursor must be a positive
integer`; it is not an ORM dictionary error. The initial `"0"` cursor is valid.

## Performance Notes

- Create a GIN index on the `tsvector` and, when fallback search matters, a GIN
  `gin_trgm_ops` index on the correct text column.
- An `OR` condition can make the planner combine indexes or choose a scan. Check
  `EXPLAIN (ANALYZE, BUFFERS)` with production-like data distributions.
- `QueryPageWithConfig` runs a count query in addition to the data query. Measure
  count and `OR` costs on large datasets.
- The `grpc-auth` cursor path fetches up to 1000 rows per batch and slices a page
  in memory. Reconsider the consumer strategy for wide rows or access patterns
  requiring pure keyset pagination.
- The implementation does not calculate rank. Ordering comes entirely from
  `Sort`; the example uses `a.id ASC`.

## Complete Example

The following compiles after the ORM dependency is available and a valid
PostgreSQL database adapter is supplied:

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
