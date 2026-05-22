### 2. 📂 Isi untuk file `pricing-service/README.md`

```markdown
# 🚚 Pricing Service - Fujiwara Express 

A robust microservice built with **Go (Golang)** to calculate shipping prices. This service implements industry-standard **Clean Architecture** and handles complex logistics logic such as volumetric weight and oversize surcharges.

## 🚀 Features
- **Clean Architecture**: Decoupled layers (Domain, Delivery, Service, Repository).
- **Volumetric Logic**: Automatically calculates weight based on dimensions ($L \times W \times H / 6000$).
- **Smart Weight Selection**: Compares actual weight vs. volumetric weight and picks the highest.
- **Oversize Surcharge**: Automatically applies a penalty if the package dimensions exceed the service limit (e.g., length > 100cm).
- **Unit Testing**: 100% logic coverage using `Gomock`.

## 📂 Project Structure
```text
pricing-service/
├── internal/
│   ├── delivery/http/    # Handler: Penerima request HTTP (API Gate)
│   ├── service/          # Service: Otak aplikasi (Logika Bisnis)
│   ├── repository/       # Repository: Akses data (In-Memory/Database)
│   └── domain/           # Domain: Model data dan Error definitions
├── main.go               # Entry point aplikasi
├── go.mod                # Dependency management
└── README.md             # Dokumentasi
🛠 PrerequisitesGo 1.20 or later.Thunder Client (VS Code Extension) or Postman.🏃 How to RunBuka terminal di folder proyek. Jalankan aplikasi:Bashgo run main.go
Aplikasi akan berjalan di http://localhost:8080.🧪 TestingUntuk menjalankan unit test dan memastikan logika perhitungan harga benar:Bashgo test ./internal/service/ -v
📍 API Documentation (Calculate Price)Menghitung harga pengiriman berdasarkan lokasi, berat, dan dimensi.URL: /calculate-priceMethod: POSTRequest Header: Content-Type: application/jsonContoh Request Body:JSON{
    "origin": "CGK",
    "destination": "BDO",
    "service_type": "REG",
    "weight": 5,
    "length": 150,
    "width": 50,
    "height": 50
}
Contoh Response (Success):JSON{
    "service_type": "REG",
    "price": 675000,
    "volumetric_weight": 62.5,
    "chargeable_weight": 62.5,
    "breakdown": {
        "base_rate": 625000,
        "oversize_surcharge": 50000
    }
}
📝 Business Logic NotesBerat Volumetrik: Jika paket besar tapi ringan, harga dihitung berdasarkan volume. Rumus: $(P \times L \times T) / 6000$.Surcharge: Jika salah satu dimensi ($P, L, T$) melebihi batas maksimal di database (default: 100cm), maka denda Oversize sebesar Rp 50.000 akan otomatis ditambahkan ke total harga.

<img width="1920" height="1080" alt="Screenshot (592)" src="https://github.com/user-attachments/assets/f7653774-22fd-4bbb-ae5c-582bb001ab9c" />
