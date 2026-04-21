# 🏠 Kost App — Manajemen Kost

Aplikasi web manajemen kost dengan fitur:
- Input data penghuni (No Kamar, Nama, No HP, Check-in)
- Harga otomatis Harian / Bulanan dari setting
- Generate invoice PDF otomatis saat check-in
- Kirim invoice PDF ke Telegram Bot
- Dashboard statistik penghuni

---

## 🛠 Teknologi

| Komponen | Teknologi |
|---|---|
| Backend / HTTP Server | **Go** (net/http stdlib) |
| Database | **SQLite 3** (via CLI) |
| PDF Generator | **Python 3** + fpdf2 |
| Telegram | **Bot API** (via urllib) |
| Frontend | Vanilla JS + CSS (dark theme) |

---

## 📦 Requirements

- **Go** 1.18+ → https://go.dev/dl/
- **Python 3** → https://python.org
- **SQLite3** CLI → `sudo apt install sqlite3`
- **fpdf2** → `pip install fpdf2`

---

## 🚀 Menjalankan Aplikasi

### Cara 1 — Script otomatis
```bash
chmod +x start.sh
./start.sh
```

### Cara 2 — Manual
```bash
# Build
go build -o kost-app .

# Jalankan
./kost-app
```

Buka browser: **http://localhost:8080**

---

## ⚙️ Konfigurasi Awal

1. Buka **Pengaturan** (sidebar kiri)
2. Isi **Nama Kost** dan **Alamat**
3. Atur **Harga Harian** dan **Harga Bulanan**
4. Isi **Telegram Bot Token** dan **Chat ID** (opsional)
5. Klik **Simpan Pengaturan**

### Setup Telegram Bot:
1. Chat @BotFather di Telegram → `/newbot`
2. Salin **Bot Token** yang diberikan
3. Dapatkan **Chat ID** via @userinfobot atau @getmyid_bot
4. Masukkan keduanya di halaman Pengaturan

---

## 📋 Alur Kerja

```
Input Data Penghuni
       ↓
Harga otomatis (Harian/Bulanan dari Setting)
       ↓
Simpan ke Database (SQLite)
       ↓
Generate Invoice PDF (Python + fpdf2)
       ↓
Kirim PDF ke Telegram Bot
       ↓
Tampil di Daftar Penghuni
```

---

## 📁 Struktur File

```
kost-app/
├── main.go          # Go HTTP server + semua handler API
├── gen_invoice.py   # Python script generate PDF invoice
├── go.mod           # Go module file
├── start.sh         # Script startup
├── kost-app         # Binary (setelah build)
├── kost.db          # Database SQLite (auto-dibuat)
├── invoices/        # Folder PDF invoice (auto-dibuat)
└── web/
    └── index.html   # Frontend SPA
```

---

## 🔌 API Endpoints

| Method | Path | Deskripsi |
|---|---|---|
| GET | `/api/settings` | Ambil konfigurasi |
| POST | `/api/settings` | Simpan konfigurasi |
| GET | `/api/penghuni` | Daftar semua penghuni |
| POST | `/api/penghuni` | Tambah penghuni baru + generate invoice |
| DELETE | `/api/penghuni/{id}` | Check-out penghuni |
| GET | `/api/invoice/{id}` | Download PDF invoice |
| POST | `/api/resend/{id}` | Kirim ulang invoice ke Telegram |

---

## 📄 Contoh Request Tambah Penghuni

```bash
curl -X POST http://localhost:8080/api/penghuni \
  -H 'Content-Type: application/json' \
  -d '{
    "no_kamar": "101",
    "nama": "Budi Santoso",
    "no_hp": "081234567890",
    "check_in": "2025-04-01",
    "tipe_harga": "bulanan"
  }'
```

Response:
```json
{
  "status": "ok",
  "id": 1,
  "harga": 800000,
  "tipe": "bulanan",
  "pdf_path": "invoices/invoice_1.pdf"
}
```
