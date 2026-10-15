# Query Builder

The main entrypoint for read queries is:

```go
q := db.UseModel(User{})
```

`UseModel` creates a query builder and sets the model table. The lower-level
`Table(model)` method also exists on `query.QueryBuilder`; `UseModel` calls it
for you.

## Select

Without `Select`, the builder selects mapped model columns.

```go
err := db.UseModel(User{}).
    Select("id", "email").
    Scan(&users)
```

`Select` also accepts raw select expressions when the string contains spaces,
dots, parentheses, `*`, or ` AS `:

```go
err := db.UseModel(User{}).
    Select("status", "COUNT(*) AS total").
    GroupBy("status").
    Scan(&rows)
```

There is no separate public `SelectExpr` method in the current implementation.

## Where

`Where` and `OrWhere` accept SQL fragments with `?` placeholders:

```go
err := db.UseModel(User{}).
    Where("status = ?", "ACTIVE").
    OrWhere("email LIKE ?", "%@example.com").
    Scan(&users)
```

Grouped conditions are supported:

```go
err := db.UseModel(User{}).
    Where("status = ?", "ACTIVE").
    WhereGroup(func(q *query.QueryBuilder) {
        q.Where("email LIKE ?", "%@example.com").
            OrWhere("email LIKE ?", "%@company.com")
    }).
    Scan(&users)
```

Use `OrWhereGroup` when the group should be connected with `OR`.

## Safe Where Operators

`WhereOp` and `OrWhereOp` provide typed operators while preserving the existing
`Where` API.

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

Available operators:

- `orm.OpEqual`
- `orm.OpNotEqual`
- `orm.OpLessThan`
- `orm.OpLessThanEqual`
- `orm.OpGreaterThan`
- `orm.OpGreaterThanEqual`
- `orm.OpLike`
- `orm.OpNotLike`

`Where` is still supported and is not deprecated. Prefer `WhereOp` when you
want typed operator constants.

## PostgreSQL Full Text Where

`WhereFullText` and `OrWhereFullText` add a PostgreSQL-only full text search
condition for a `tsvector` column:

```go
builder, err := db.UseModel(Profile{}).
    Join("user_profile_search ups", "ups.profile_id = profiles.id").
    WhereFullText("ups.fts_keyword", keyword)
if err != nil {
    return err
}
```

The generated condition is:

```sql
ups.fts_keyword @@ websearch_to_tsquery('simple', ?)
```

Placeholders are rebound by dialect, so PostgreSQL DryRun output uses `$1`.
MySQL and Oracle return `ErrUnsupportedSearchModeForDialect` for this helper.
Use `WhereOp` only for normal comparison operators; full text search is not
represented as `=`, `<`, `LIKE`, or `IN`.

## In Lists

```go
err := db.UseModel(User{}).
    WhereIn("status", []string{"ACTIVE", "PENDING"}).
    WhereNotIn("email", []string{"blocked@example.com"}).
    Scan(&users)
```

`OrWhereIn` and `OrWhereNotIn` are also available.

## Joins

The builder supports inner, left, and right joins:

```go
err := db.UseModel(User{}).
    Join("roles", "roles.user_id = users.id").
    LeftJoin("profiles", "profiles.user_id = users.id").
    RightJoin("accounts", "accounts.user_id = users.id").
    Scan(&users)
```

Join table and `ON` expressions are explicit strings.

## Ordering And Limits

```go
err := db.UseModel(User{}).
    OrderBy("created_at DESC").
    Limit(20).
    Offset(40).
    Scan(&users)
```

`Limit` ignores values less than or equal to zero. `Offset` ignores negative
values.

## Distinct, Group By, And Having

```go
err := db.UseModel(User{}).
    Select("status", "COUNT(*) AS total").
    Distinct().
    GroupBy("status").
    Having("COUNT(*) > ?", 1).
    OrHaving("COUNT(*) = ?", 0).
    Scan(&rows)
```

Paginator count queries use a wrapped count query for complex shapes such as
`DISTINCT`, `GROUP BY`, `HAVING`, raw select expressions, and alias selects.

## First

`First` applies `LIMIT 1`, runs the query, and scans one row into the
destination:

```go
var user User
err := db.UseModel(User{}).
    Where("email = ?", email).
    First(&user)
```

## Composable subqueries

Compose child builders on the same transaction. Children are snapshots: later
builder edits do not change a previously composed parent. Rendering uses the
parent dialect, placeholder mode and identifier configuration, including nested
arguments after UPDATE SET values. A nil/invalid child or invalid derived alias
fails before execution. Derived aliases use a simple identifier without AS.

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

`DryRun`, count and pagination preserve child/parent builders. `WhereExists`,
`WhereNotExists` and `WhereInQuery` also work with `Updates` and `DryRunUpdates`.
Expressions (columns, projections, correlations and ON) remain trusted SQL
structure; bind request values and whitelist identifiers as with existing Where.
Neither a complete SELECT string nor placeholder rewriting is needed in callers.
