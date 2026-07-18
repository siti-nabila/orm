# ORM Documentation

This documentation describes the public API that exists in this repository.
The ORM is intentionally small: SQL is explicit, transaction ownership stays
with the caller, and dialect behavior is kept visible.

## Guides

- [Getting Started](getting-started.md)
- [Query Builder](query-builder.md)
- [Create, Update, and Bulk Writes](create-update.md)
- [Scanning Results](scan.md)
- [Pagination](pagination.md)
- [Dry Run](dry-run.md)
- [Transactions and Locking](transactions.md)
- [Dialect Notes](dialects.md)
- [Examples](examples.md)

## Model Used In Examples

```go
type User struct {
    ID        int64  `sql:"column:id;primaryKey"`
    Name      string `sql:"column:name"`
    Email     string `sql:"column:email"`
    Status    string `sql:"column:status"`
    CreatedAt string `sql:"column:created_at"`
}

func (User) TableName() string {
    return "users"
}
```

Struct fields are mapped with the `sql` tag. Use `column:name` to set the
database column name, `primaryKey` to mark the primary key field, and `where` to
mark a fallback update condition for models without a primary key. Use
`sql:"-"` to skip a field.

```go
type ApprovalLogUpdate struct {
    ReferenceID string `sql:"column:reference_id;where"`
    Flag        int    `sql:"column:flag"`
}
```

See [Create, Update, and Bulk Writes](create-update.md) for condition precedence,
zero-value behavior, chained updates, and safety rules.
