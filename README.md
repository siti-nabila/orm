# 🧩 ORM Native Go (Lightweight)

ORM Native Go adalah library sederhana untuk membantu melakukan operasi database seperti **INSERT** secara dinamis menggunakan struct di Go.

Library ini dirancang:

* 🚀 Lightweight (tanpa dependency berat)
* 🧠 Simple & explicit (tidak magic berlebihan)
* 🔧 Mendukung multiple SQL driver (Postgres, MySQL, Oracle – planned)

---

## ✨ Features

* ✅ Generate query INSERT otomatis dari struct
* ✅ Support custom tag (`sql:"column:..."`)
* ✅ Auto mapping struct → column
* ✅ Support primary key auto assign (LastInsertId)
* ⚙️ Configurable (snake_case optional)
* 🧪 Compatible dengan `sqlmock` (untuk testing)

---

## 📦 Installation

```bash
go get github.com/siti-nabila/orm
```

---

## 🏗️ Structure Example

```go
type User struct {
	ID       uint64  `sql:"column:id;primaryKey"`
	Email    string  `sql:"column:email"`
	Password string  `sql:"column:password"`
}

func (User) TableName() string {
	return "users"
}
```

---

## ⚙️ Configuration

```go
orm.SetConfig(orm.Config{
	UseSnakeCase: true,
})
```

---

## 🚀 Usage (Repository Layer)

> ⚠️ Disarankan digunakan di repository layer, bukan langsung di main

```go
func (r *userRepository) CreateUser(user *User) error {
	return orm.Create(r.db, user)
}
```

---

## 🔄 Behavior

### Insert Flow

1. Parse struct menggunakan reflection
2. Ambil tag `sql:"column:..."` sebagai nama column
3. Generate query:

```sql
INSERT INTO users(email, password) VALUES($1, $2)
```

4. Execute query
5. Jika ada primary key (int/uint), akan auto di-set dari `LastInsertId`

---

## 🧠 Tag Support

| Tag           | Keterangan              |
| ------------- | ----------------------- |
| `column:name` | Nama column di database |
| `primaryKey`  | Menandakan primary key  |

Contoh:

```go
ID uint64 `sql:"column:id;primaryKey"`
```

---

## 🔢 Supported Types

* int, int64
* uint, uint64
* string
* pointer (*uint64, dll)

---

## ⚠️ Notes

* Field tanpa tag akan:

  * pakai nama field (atau snake_case jika diaktifkan)
* Field dengan nilai zero tetap akan ikut di insert
* Belum support:

  * UPDATE
  * DELETE
  * SELECT (coming soon)
  * PAGINATION

---

## 🧪 Testing

Library ini bisa digunakan dengan `sqlmock`:

```go
mock.ExpectExec("INSERT INTO users").
	WithArgs("test@mail.com", "password").
	WillReturnResult(sqlmock.NewResult(1, 1))
```

---

## 🛣️ Roadmap

* [ ] Insert batch
* [ ] Update builder
* [ ] Select query builder
* [ ] Where clause support
* [ ] Transaction support (multi DB)
* [ ] Hook (BeforeInsert, AfterInsert)

---

## 💡 Philosophy

Library ini dibuat untuk:

> "Memberikan kontrol penuh ke developer tanpa kehilangan kenyamanan"

Bukan untuk menggantikan SQL, tapi membantu mengurangi boilerplate.

---

## 👩‍💻 Author

Made with ❤️ for learning & production use.

---

## 📄 License

MIT License
