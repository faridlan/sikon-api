# 📚 SIKOn API (Sistem Integrasi Konveksi Online)

SIKOn (Sistem Integrasi Konveksi Online) adalah sebuah sistem Enterprise Resource Planning (ERP) tahap MVP (Minimum Viable Product) yang dirancang khusus untuk memodernisasi manajemen bisnis konveksi.

SIKOn API menangani inti bisnis konveksi mulai dari manajemen kategori, produk, pencatatan data pelanggan, pendaftaran pengguna (Sales/Admin), manajemen multi-rekening bank, hingga proses pemesanan (Order) dan pelacakan pembayaran (Payment) baik secara uang muka (DP) maupun pelunasan (Settlement).

---

# 🛠️ Tech Stack & Architecture

Proyek ini dibangun dengan menerapkan Clean Architecture berlapis:

```text
Domain ➔ Repository ➔ Usecase ➔ Handler ➔ Router
```

Pendekatan ini memisahkan business logic dari delivery mechanism untuk memastikan sistem sangat terstruktur, mudah diuji, dan scalable untuk dijadikan SaaS (Software as a Service) di masa depan.

## Teknologi Utama

- **Bahasa Pemrograman:** Golang (Go)
- **Web Framework:** Go Fiber v2
- **Database:** PostgreSQL
- **ORM:** GORM
- **Database Migration:** Golang-Migrate (dengan Embed iofs)
- **Unit Testing:** Testify & Mockery
- **API Documentation:** Swaggo (Swagger UI)

---

# 📦 Daftar Paket (Dependencies) Utama

Berikut adalah package utama yang menyokong sistem ini:

- `github.com/gofiber/fiber/v2` — Core HTTP Server
- `gorm.io/gorm` & `gorm.io/driver/postgres` — Database ORM & Driver
- `github.com/golang-migrate/migrate/v4` — Eksekusi Migrasi SQL
- `golang.org/x/crypto/bcrypt` — Hashing Password User
- `github.com/go-playground/validator/v10` — Validasi Data Request/DTO
- `github.com/google/uuid` — Generate ID unik
- `github.com/joho/godotenv` — Environment Config Loader
- `github.com/swaggo/swag` & `github.com/gofiber/swagger` — Dokumentasi API interaktif
- `github.com/stretchr/testify` & `github.com/vektra/mockery` — Testing Toolkit

---

# ⚙️ Konfigurasi Environment (.env)

Untuk menjalankan SIKOn API, Anda wajib membuat file `.env` di root proyek.

```env
# Application Server Config
APP_PORT=8080
FRONTEND_URL=http://localhost:3000

# Database Config (PostgreSQL)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=rahasia
DB_NAME=sikon_erp
DB_URL=postgres://postgres:rahasia@localhost:5432/sikon_erp?sslmode=disable

# Swagger Setup
SWAGGER_HOST=localhost:8080
```

---

# 🗺️ Perjalanan Pengembangan SIKOn API (Dari Awal - Akhir)

Pengembangan sistem ini mengikuti disiplin **Domain-Driven Design** dan **Clean Architecture**.

## 1. Desain Flow Bisnis & Database

### Analisis Kebutuhan
Merancang flow konveksi:

```text
Produk ➔ Customer ➔ Order ➔ Payment DP/Lunas
```

### Skema Database
Menggunakan UUID untuk keamanan.

Entitas utama:

- Users
- Categories
- Products
- Customers
- Bank_Accounts
- Orders
- Order_Items
- Payments

---

## 2. Setup Layer DOMAIN (Entitas & Kontrak)

- Mendefinisikan struct entitas murni Go di folder `internal/domain`
- Membuat tipe data kustom (Enum) untuk:
  - Peran pengguna
  - Status pesanan
  - Tipe pembayaran
- Mendefinisikan interface (kontrak) untuk:
  - Repository (interaksi database)
  - Usecase (logika bisnis)

### Refactoring
Menambahkan Input Struct seperti `ProductCreateInput` untuk memisahkan entitas database dari input request, sehingga mencegah over-posting.

---

## 3. Setup Error Handling & Utilities Terpusat

- Membuat `AppError` kustom di `domain/errors.go`
- Membuat standarisasi balasan API (Response Formatter) melalui fungsi:
  - `utils.SendSuccess`
  - `utils.HandleDomainError`
- Menginisialisasi `go-playground/validator` di dalam utils untuk menerjemahkan error validasi ke dalam bahasa Indonesia

Contoh:

```text
"email tidak boleh kosong"
```

---

## 4. Setup Layer USECASE (Business Logic)

Mengimplementasikan fungsi logika bisnis yang patuh pada kontrak Domain.

### Order Usecase

Fitur utama:

- Memverifikasi ID Produk
- Menghitung otomatis total amount order
- Mencetak otomatis nomor resi/nota

Contoh format:

```text
ORD-YYYYMMDD-XXXX
```

### Payment Usecase

Fitur utama:

- Mengecek validitas Bank dan Order
- Menjumlahkan riwayat pembayaran
- Mencegah pembayaran double
- Mengubah otomatis status pesanan:

```text
DP ➔ Lunas
```

Jika total pembayaran menyamai grand total tagihan.

---

## 5. Setup DTO & Layer HANDLER (Delivery/HTTP)

### DTO (Data Transfer Object)

- Memisahkan format data request/response (JSON) dari entitas inti
- Menambahkan tag `example:""` untuk dokumentasi Swagger

### Handler

Tanggung jawab handler:

- Menangani HTTP Request via Fiber
- Parsing DTO
- Validasi sintaksis:
  - UUID
  - Format string
- Meneruskan data bersih ke layer Usecase

---

## 6. Swagger & Routing

### Swagger Annotation

Menambahkan anotasi seperti:

```go
// @Summary
// @Param
// @Success
```

Pada setiap fungsi handler.

### Dependency Injection

Menerapkan Struct Dependency Injection pada `router.go` agar pengelolaan endpoint tetap rapi dan menghindari parameter fungsi yang terlalu panjang.

---

## 7. Unit Testing

- Generate mock dari setiap interface Repository menggunakan Mockery
- Menulis unit test menggunakan Testify
- Menguji seluruh logika bisnis (Usecase)

### Fokus Pengujian

- Percabangan error
- Validasi data
- Kalkulasi biaya
- Skenario customer tidak ditemukan
- Validasi pembayaran

---

## 8. The Main Wiring (Server Initialization)

### Konfigurasi Infrastruktur

- Koneksi GORM dengan Connection Pooling
- Migrasi otomatis berbasis file Embed
- Logger terpusat menggunakan `slog`

### Dependency Injection Utama

Mengikat seluruh dependency:

```text
Repository ➔ Usecase ➔ Handler
```

Di dalam fungsi `main()`.

### Server Runtime

- Menjalankan server Fiber
- Mendukung Graceful Shutdown

---

# 🚀 Panduan Menggunakan Proyek Ini (How to Run)

## Persiapan Sistem (Prerequisites)

Pastikan sistem Anda memiliki:

- Golang versi 1.21 atau lebih baru
- PostgreSQL aktif dan berjalan
- Golang-Migrate CLI (opsional)
- Swaggo CLI

---

## Langkah Menjalankan Aplikasi

### 1. Clone Repository

```bash
git clone https://github.com/username/sikon-api.git
cd sikon-api
```

---

### 2. Install Dependencies

```bash
go mod tidy
```

---

### 3. Konfigurasi Environment

- Salin file `.env.example` (jika tersedia)
- Atau buat file `.env` baru
- Isi konfigurasi PostgreSQL sesuai kebutuhan
- Pastikan database `sikon_erp` sudah dibuat

---

### 4. Generate Swagger Documentation

Jalankan perintah berikut agar dokumentasi Swagger selalu sinkron dengan kode:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/main.go --parseDependency --parseInternal
```

---

### 5. Generate Mocks

Jika ada perubahan interface atau ingin menjalankan test:

```bash
go install github.com/vektra/mockery/v2@v2.42.1
mockery --dir=internal/domain --all --output=internal/domain/mocks --outpkg=mocks
```

---

### 6. Jalankan Unit Testing

Pastikan seluruh business logic berjalan dengan baik:

```bash
go test ./internal/usecase/... -v
```

---

### 7. Jalankan Server

```bash
go run cmd/main.go
```

Server akan berjalan di:

```text
http://localhost:8080
```

Migrasi database akan otomatis dijalankan jika `DB_URL` terkonfigurasi.

---

# 📖 Mengakses Dokumentasi (Swagger UI)

Setelah server berjalan, dokumentasi API dapat diakses melalui browser:

```text
http://localhost:8080/swagger/
```

Melalui Swagger UI, Anda dapat:

- Melihat seluruh endpoint API
- Mencoba request secara langsung (`Try it out`)
- Melihat struktur request & response
- Menguji validasi endpoint secara interaktif

---

# ✅ Penutup

SIKOn API dirancang sebagai fondasi ERP konveksi modern dengan fokus pada:

- Arsitektur bersih dan scalable
- Pemisahan tanggung jawab yang jelas
- Konsistensi response API
- Kemudahan testing
- Kesiapan menuju SaaS multi-tenant di masa depan

Dengan kombinasi Clean Architecture, Domain-Driven Design, dan tooling modern di ekosistem Golang, proyek ini menjadi fondasi yang kuat untuk pengembangan sistem ERP konveksi skala kecil hingga enterprise.

