### 3. 📂 Isi untuk file `api-gateway/README.md`

```markdown
# 🚪 API Gateway - Fujiwara Express

API Gateway ini bertindak sebagai pintu masuk tunggal (*Single Entry Point*) untuk seluruh ekosistem *microservices* logistik Fujiwara Express. Dibangun menggunakan **Go (Golang)** dengan memanfaatkan `httputil.ReverseProxy`.

---

## 🎯 Konsep & Arsitektur

Alih-alih membiarkan klien (aplikasi mobile/web) menghafal banyak port untuk setiap layanan (8080 untuk Pricing, 8081 untuk Warehouse), API Gateway menyederhanakannya. Klien cukup menembak ke **Port 8000**, dan Gateway yang akan meneruskan (*routing*) permintaan tersebut ke layanan yang tepat di belakang layar.

```text
[Klien / Thunder Client] 
       │ (Port 8000)
       ▼
[ API GATEWAY ]
       │
       ├─ Jika `/pricing/*`   ➔ Diteruskan ke http://localhost:8080 (Pricing Service)
       └─ Jika `/warehouse/*` ➔ Diteruskan ke http://localhost:8081 (Warehouse Service)
🚀 Fitur Utama
Reverse Proxy: Menggunakan httputil.NewSingleHostReverseProxy bawaan Go untuk performa routing jaringan yang sangat cepat tanpa hambatan.

Path Stripping: Otomatis menghapus awalan /pricing atau /warehouse sebelum meneruskan permintaan ke layanan di belakangnya agar URL tetap bersih.

⚙️ Cara Menjalankan Sistem
Karena ini adalah gateway, layanan di belakangnya wajib dinyalakan terlebih dahulu.

1. Buka Terminal 1, jalankan Pricing Service:

Bash
cd pricing-service
go run main.go
2. Buka Terminal 2, jalankan Warehouse Service:

Bash
cd warehouse-service
go run main.go
3. Buka Terminal 3, jalankan API Gateway:

Bash
cd api-gateway
go run main.go
API Gateway akan menyala di http://localhost:8000.

📡 Dokumentasi Endpoint (Via Gateway)
Semua request sekarang dilakukan melalui Port 8000.

1. Pricing Service
Gunakan awalan /pricing untuk mengakses layanan harga.

Hitung Harga: POST http://localhost:8000/pricing/calculate-price

2. Warehouse Service
Gunakan awalan /warehouse untuk mengakses layanan gudang.

Barang Masuk: POST http://localhost:8000/warehouse/receive

Cek Stok: GET http://localhost:8000/warehouse/check-stock?id=BRG-001

<<<<<<< HEAD
Barang Keluar: POST http://localhost:8000/warehouse/dispatch
=======
Barang Keluar: POST http://localhost:8000/warehouse/dispatch

<img width="1920" height="1080" alt="Screenshot (608)" src="https://github.com/user-attachments/assets/cc5e0dc1-9058-4357-aad8-ab29a57dd520" />
<img width="1920" height="1080" alt="Screenshot (609)" src="https://github.com/user-attachments/assets/c8025aa2-e347-4a78-8bd4-4431489272a2" />



>>>>>>> f2c9bf04b4800aae72df34354fd7d67e7c64c033
