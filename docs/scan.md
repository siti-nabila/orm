# Scanning Results

`Scan` and `First` are query builder methods.

## Scan Into A Struct

```go
var user User

err := db.UseModel(User{}).
    Where("email = ?", email).
    First(&user)
```

`First` applies `LIMIT 1` and scans one row. A missing row is normalized through
the repository error normalization layer.

## Scan Into A Slice Of Struct

```go
var users []User

err := db.UseModel(User{}).
    Where("status = ?", "ACTIVE").
    OrderBy("created_at DESC").
    Scan(&users)
```

The destination must be a pointer to a slice.

## Scan Primitive Values

Primitive scans are supported when the query returns exactly one column.

```go
var emails []string

err := db.UseModel(User{}).
    Select("email").
    Where("status = ?", "ACTIVE").
    Scan(&emails)
```

For one primitive value:

```go
var total int64

err := db.UseModel(User{}).
    Select("COUNT(*)").
    First(&total)
```

Primitive scan requires a single selected column.

## Dialect-Specific Scan Handling

The scanner has shared handling for integer, unsigned integer, float, boolean,
`[]byte`, and nullable string pointer fields for MySQL and Oracle numeric or
string-like driver values.

PostgreSQL has additional handling for array fields including `[]string`,
`[]int`, `[]int64`, `[]uint32`, and `[]uint64`.

The implementation also performs overflow checks for numeric conversions.
