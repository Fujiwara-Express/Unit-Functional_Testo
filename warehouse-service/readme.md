# 📦 Warehouse Service - Fujiwara Express

Warehouse Service adalah *microservice* berbasis **Go (Golang)** yang menangani manajemen inventaris (stok) logistik. Layanan ini dirancang menggunakan standar industri **Clean Architecture** untuk memastikan kode mudah dirawat (*maintainable*), mudah dites (*testable*), dan terukur (*scalable*).

---

## 🧠 Konsep Utama yang Digunakan

Proyek ini tidak sekadar membuat aplikasi yang "bisa jalan", tetapi menerapkan tiga konsep rekayasa perangkat lunak tingkat lanjut:

### 1. Clean Architecture (Arsitektur 4 Lapis)

Kode tidak dicampur aduk di satu tempat (`main.go`), melainkan dibagi menjadi 4 "ruangan" dengan tugas spesifik:
* **Domain Layer (`internal/domain`)**: Cetakan dasar aplikasi. Mendefinisikan bentuk barang (`Item`) dan bentuk permintaan (`StockRequest`). Tidak bergantung pada teknologi luar.
* **Repository Layer (`internal/repository`)**: "Rak Gudang" tempat menyimpan data. Menggunakan antarmuka (*interface*) agar kedepannya bisa dengan mudah diganti dari *In-Memory* ke PostgreSQL atau MongoDB tanpa menyentuh logika bisnis.
* **Service Layer (`internal/service`)**: "Kepala Gudang" (Logika Bisnis). Menangani aturan ketat seperti: menolak barang keluar jika stok tidak cukup, atau otomatis menjumlahkan kuantitas jika ID barang yang sama didaftarkan lagi.
* **Delivery Layer (`internal/delivery/http`)**: "Resepsionis" atau API Gate. Bertugas menerima *request* HTTP dari luar (seperti dari Postman/Thunder Client), memastikan format JSON benar, lalu meneruskannya ke Service.

### 2. Keamanan Data (Concurrency Control dengan Mutex)
Karena Go sangat cepat dan mendukung eksekusi paralel (*Goroutines*), penyimpanan In-Memory kita sangat rentan terhadap **Race Condition** (kondisi di mana dua proses mencoba mengubah data yang sama di saat bersamaan).
Sistem ini menggunakan `sync.RWMutex` sebagai "Gembok Sakti":
- `mu.Lock()` / `mu.Unlock()`: Digunakan saat menambah/mengurangi stok agar hanya ada 1 antrean yang bisa merubah data di satu waktu.
- `mu.RLock()` / `mu.RUnlock()`: Digunakan saat hanya membaca stok, sehingga banyak proses bisa membaca bersamaan tanpa performa melambat.

### 3. Microservices Architecture
Gudang ini berjalan di **Port 8081**, terpisah dari Pricing Service (Port 8080). Jika layanan harga (*Pricing*) mengalami *error* atau mati, layanan gudang (*Warehouse*) akan tetap beroperasi dengan normal tanpa gangguan.

---

## 📂 Struktur Folder Proyek

```text
warehouse-service/
├── internal/
│   ├── delivery/http/    # HTTP Handlers (Resepsionis)
│   ├── service/          # Business Logic (Kepala Gudang)
│   ├── repository/       # Data Access (Rak Penyimpanan In-Memory)
│   └── domain/           # Models and Errors (Bentuk Barang)
├── main.go               # Entry Point (Saklar Utama)
├── go.mod                # Go Modules (KTP Proyek)
└── README.md             # Dokumentasi
🏳️ API Testing Guide (Via Port 8081)
Tes 1: Barang Masuk (Receive Item)
Mencatat barang baru atau menambah stok barang yang sudah ada.

URL: http://localhost:8081/receive

Method: POST

Headers: Content-Type: application/json

Request Body:

JSON
{
    "item_id": "BRG-001",
    "name": "Laptop Gaming",
    "quantity": 50,
    "location": "Rak-A1"
}
Ekspektasi Output: {"message": "Barang berhasil masuk gudang!"} (Status: 201 Created)

Tes 2: Cek Ketersediaan Stok (Check Stock)
Melihat data spesifik suatu barang berdasarkan ID untuk memastikan barang benar-benar sudah masuk.

URL: http://localhost:8081/check-stock?id=BRG-001

Method: GET

Ekspektasi Output: (Status: 200 OK)

JSON
{
    "item_id": "BRG-001",
    "name": "Laptop Gaming",
    "quantity": 50,
    "location": "Rak-A1"
}
Tes 3: Keluarkan Barang (Dispatch Item)
Mengurangi stok barang dari gudang. Sistem akan otomatis memvalidasi dan menolak jika stok tidak mencukupi.

URL: http://localhost:8081/dispatch

Method: POST

Headers: Content-Type: application/json

Request Body:

JSON
{
    "item_id": "BRG-001",
    "quantity": 5
}
Ekspektasi Output: {"message": "Barang berhasil dikeluarkan!"} (Status: 200 OK)