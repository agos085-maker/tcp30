package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ─────────────────────────────────────────────
// CGO SQLite3 — pure Go driver via os/exec sqlite3 CLI
// We use database/sql with a lightweight embedded driver
// ─────────────────────────────────────────────

var db *Database

type Database struct {
	path string
}

func NewDatabase(path string) *Database {
	return &Database{path: path}
}

func (d *Database) Exec(query string, args ...interface{}) error {
	q := interpolate(query, args...)
	cmd := exec.Command("sqlite3", d.path, q)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sqlite3 exec error: %v | %s", err, out)
	}
	return nil
}

func (d *Database) Query(query string, args ...interface{}) ([]map[string]string, error) {
	q := interpolate(query, args...)
	cmd := exec.Command("sqlite3", "-separator", "\x1F", "-header", d.path, q)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("sqlite3 query error: %v", err)
	}
	return parseResult(string(out)), nil
}

func interpolate(query string, args ...interface{}) string {
	for _, arg := range args {
		var s string
		switch v := arg.(type) {
		case string:
			s = "'" + strings.ReplaceAll(v, "'", "''") + "'"
		case int:
			s = strconv.Itoa(v)
		case int64:
			s = strconv.FormatInt(v, 10)
		case float64:
			s = strconv.FormatFloat(v, 'f', 2, 64)
		case nil:
			s = "NULL"
		default:
			s = fmt.Sprintf("'%v'", v)
		}
		query = strings.Replace(query, "?", s, 1)
	}
	return query
}

func parseResult(raw string) []map[string]string {
	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	if len(lines) < 2 {
		return nil
	}
	headers := strings.Split(lines[0], "\x1F")
	var rows []map[string]string
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		cols := strings.Split(line, "\x1F")
		row := make(map[string]string)
		for i, h := range headers {
			if i < len(cols) {
				row[h] = cols[i]
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// ─────────────────────────────────────────────
// Models
// ─────────────────────────────────────────────

type Settings struct {
	NamaKost       string  `json:"nama_kost"`
	Alamat         string  `json:"alamat"`
	TelegramToken  string  `json:"telegram_token"`
	TelegramChatID string  `json:"telegram_chat_id"`
	HargaHarian    float64 `json:"harga_harian"`
	HargaBulanan   float64 `json:"harga_bulanan"`
}

type Penghuni struct {
	ID           int     `json:"id"`
	NoKamar      string  `json:"no_kamar"`
	Nama         string  `json:"nama"`
	NoHP         string  `json:"no_hp"`
	CheckIn      string  `json:"check_in"`
	TipeHarga    string  `json:"tipe_harga"` // harian / bulanan
	Harga        float64 `json:"harga"`
	Status       string  `json:"status"` // aktif / keluar
	CreatedAt    string  `json:"created_at"`
}

// ─────────────────────────────────────────────
// Init DB
// ─────────────────────────────────────────────

func initDB() {
	db = NewDatabase("kost.db")
	db.Exec(`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS penghuni (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		no_kamar TEXT NOT NULL,
		nama TEXT NOT NULL,
		no_hp TEXT,
		check_in TEXT NOT NULL,
		tipe_harga TEXT NOT NULL DEFAULT 'bulanan',
		harga REAL NOT NULL,
		status TEXT NOT NULL DEFAULT 'aktif',
		created_at TEXT DEFAULT (datetime('now','localtime'))
	)`)
	// seed default settings
	db.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES ('nama_kost', 'Kost Sejahtera')`)
	db.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES ('alamat', 'Jl. Mawar No. 1')`)
	db.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES ('harga_harian', '75000')`)
	db.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES ('harga_bulanan', '800000')`)
	db.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES ('telegram_token', '')`)
	db.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES ('telegram_chat_id', '')`)
	os.MkdirAll("invoices", 0755)
	log.Println("Database initialized")
}

func getSettings() Settings {
	rows, _ := db.Query(`SELECT key, value FROM settings`)
	m := make(map[string]string)
	for _, r := range rows {
		m[r["key"]] = r["value"]
	}
	hh, _ := strconv.ParseFloat(m["harga_harian"], 64)
	hb, _ := strconv.ParseFloat(m["harga_bulanan"], 64)
	return Settings{
		NamaKost:       m["nama_kost"],
		Alamat:         m["alamat"],
		TelegramToken:  m["telegram_token"],
		TelegramChatID: m["telegram_chat_id"],
		HargaHarian:    hh,
		HargaBulanan:   hb,
	}
}

// ─────────────────────────────────────────────
// HTTP Handlers
// ─────────────────────────────────────────────

func jsonResp(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

// GET /api/settings
// POST /api/settings
func handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		jsonResp(w, 200, getSettings())
	case "POST":
		var s Settings
		json.NewDecoder(r.Body).Decode(&s)
		db.Exec(`INSERT OR REPLACE INTO settings (key,value) VALUES ('nama_kost', ?)`, s.NamaKost)
		db.Exec(`INSERT OR REPLACE INTO settings (key,value) VALUES ('alamat', ?)`, s.Alamat)
		db.Exec(`INSERT OR REPLACE INTO settings (key,value) VALUES ('telegram_token', ?)`, s.TelegramToken)
		db.Exec(`INSERT OR REPLACE INTO settings (key,value) VALUES ('telegram_chat_id', ?)`, s.TelegramChatID)
		db.Exec(`INSERT OR REPLACE INTO settings (key,value) VALUES ('harga_harian', ?)`, fmt.Sprintf("%.0f", s.HargaHarian))
		db.Exec(`INSERT OR REPLACE INTO settings (key,value) VALUES ('harga_bulanan', ?)`, fmt.Sprintf("%.0f", s.HargaBulanan))
		jsonResp(w, 200, map[string]string{"status": "ok"})
	}
}

// GET /api/penghuni
func handleListPenghuni(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id,no_kamar,nama,no_hp,check_in,tipe_harga,harga,status,created_at FROM penghuni ORDER BY id DESC`)
	if err != nil {
		jsonResp(w, 500, map[string]string{"error": err.Error()})
		return
	}
	var result []Penghuni
	for _, row := range rows {
		id, _ := strconv.Atoi(row["id"])
		h, _ := strconv.ParseFloat(row["harga"], 64)
		result = append(result, Penghuni{
			ID: id, NoKamar: row["no_kamar"], Nama: row["nama"],
			NoHP: row["no_hp"], CheckIn: row["check_in"],
			TipeHarga: row["tipe_harga"], Harga: h,
			Status: row["status"], CreatedAt: row["created_at"],
		})
	}
	if result == nil {
		result = []Penghuni{}
	}
	jsonResp(w, 200, result)
}

// POST /api/penghuni
func handleAddPenghuni(w http.ResponseWriter, r *http.Request) {
	var p Penghuni
	json.NewDecoder(r.Body).Decode(&p)

	s := getSettings()
	if p.TipeHarga == "harian" {
		p.Harga = s.HargaHarian
	} else {
		p.Harga = s.HargaBulanan
		p.TipeHarga = "bulanan"
	}

	// INSERT then immediately fetch the new ID in same sqlite3 process
	insertRows, err := db.Query(
		`INSERT INTO penghuni (no_kamar,nama,no_hp,check_in,tipe_harga,harga,status) VALUES (?,?,?,?,?,?,?); SELECT last_insert_rowid() as id`,
		p.NoKamar, p.Nama, p.NoHP, p.CheckIn, p.TipeHarga, p.Harga, "aktif")
	if err != nil {
		jsonResp(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if len(insertRows) > 0 {
		p.ID, _ = strconv.Atoi(insertRows[0]["id"])
	}
	// Fallback: query by unique fields if ID still 0
	if p.ID == 0 {
		rows2, _ := db.Query(`SELECT id FROM penghuni WHERE no_kamar=? AND nama=? ORDER BY id DESC LIMIT 1`, p.NoKamar, p.Nama)
		if len(rows2) > 0 {
			p.ID, _ = strconv.Atoi(rows2[0]["id"])
		}
	}

	// Generate PDF invoice
	pdfPath, err := generateInvoicePDF(p, s)
	if err != nil {
		log.Printf("PDF error: %v", err)
		jsonResp(w, 200, map[string]interface{}{"status": "ok", "id": p.ID, "pdf_error": err.Error()})
		return
	}

	// Send to Telegram
	telegramErr := ""
	if s.TelegramToken != "" && s.TelegramChatID != "" {
		if err := sendTelegram(s, p, pdfPath); err != nil {
			telegramErr = err.Error()
			log.Printf("Telegram error: %v", err)
		}
	}

	resp := map[string]interface{}{
		"status":    "ok",
		"id":        p.ID,
		"pdf_path":  pdfPath,
		"harga":     p.Harga,
		"tipe":      p.TipeHarga,
	}
	if telegramErr != "" {
		resp["telegram_error"] = telegramErr
	}
	jsonResp(w, 200, resp)
}

// DELETE /api/penghuni/{id}
func handleDeletePenghuni(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	id := parts[len(parts)-1]
	db.Exec(`UPDATE penghuni SET status='keluar' WHERE id=?`, id)
	jsonResp(w, 200, map[string]string{"status": "ok"})
}

// GET /api/invoice/{id}  -> serve PDF
func handleInvoice(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	id := parts[len(parts)-1]
	rows, err := db.Query(`SELECT id,no_kamar,nama,no_hp,check_in,tipe_harga,harga,status,created_at FROM penghuni WHERE id=?`, id)
	if err != nil || len(rows) == 0 {
		http.Error(w, "Not found", 404)
		return
	}
	row := rows[0]
	h, _ := strconv.ParseFloat(row["harga"], 64)
	pid, _ := strconv.Atoi(row["id"])
	p := Penghuni{ID: pid, NoKamar: row["no_kamar"], Nama: row["nama"],
		NoHP: row["no_hp"], CheckIn: row["check_in"],
		TipeHarga: row["tipe_harga"], Harga: h, Status: row["status"]}
	s := getSettings()
	pdfPath, err := generateInvoicePDF(p, s)
	if err != nil {
		http.Error(w, "PDF error: "+err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="invoice_%s.pdf"`, id))
	f, _ := os.Open(pdfPath)
	defer f.Close()
	io.Copy(w, f)
}

// POST /api/resend/{id}  -> resend telegram
func handleResend(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	id := parts[len(parts)-1]
	rows, _ := db.Query(`SELECT id,no_kamar,nama,no_hp,check_in,tipe_harga,harga FROM penghuni WHERE id=?`, id)
	if len(rows) == 0 {
		jsonResp(w, 404, map[string]string{"error": "not found"})
		return
	}
	row := rows[0]
	h, _ := strconv.ParseFloat(row["harga"], 64)
	pid, _ := strconv.Atoi(row["id"])
	p := Penghuni{ID: pid, NoKamar: row["no_kamar"], Nama: row["nama"],
		NoHP: row["no_hp"], CheckIn: row["check_in"],
		TipeHarga: row["tipe_harga"], Harga: h}
	s := getSettings()
	pdfPath, err := generateInvoicePDF(p, s)
	if err != nil {
		jsonResp(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if err := sendTelegram(s, p, pdfPath); err != nil {
		jsonResp(w, 500, map[string]string{"error": err.Error()})
		return
	}
	jsonResp(w, 200, map[string]string{"status": "ok"})
}

// ─────────────────────────────────────────────
// PDF Generation via Python
// ─────────────────────────────────────────────

func generateInvoicePDF(p Penghuni, s Settings) (string, error) {
	pdfPath := filepath.Join("invoices", fmt.Sprintf("invoice_%d.pdf", p.ID))
	data := map[string]interface{}{
		"id": p.ID, "no_kamar": p.NoKamar, "nama": p.Nama,
		"no_hp": p.NoHP, "check_in": p.CheckIn,
		"tipe_harga": p.TipeHarga, "harga": p.Harga,
		"nama_kost": s.NamaKost, "alamat": s.Alamat,
		"tanggal": time.Now().Format("02 January 2006"),
		"pdf_path": pdfPath,
	}
	jsonData, _ := json.Marshal(data)
	cmd := exec.Command("python3", "gen_invoice.py", string(jsonData))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, out)
	}
	return pdfPath, nil
}

// ─────────────────────────────────────────────
// Telegram
// ─────────────────────────────────────────────

func sendTelegram(s Settings, p Penghuni, pdfPath string) error {
	// Send document via multipart
	caption := fmt.Sprintf(
		"🏠 *%s*\n\n📄 Invoice Check-In\n👤 Nama: %s\n🚪 Kamar: %s\n📅 Check-in: %s\n💰 Tipe: %s\n💵 Harga: Rp %.0f",
		s.NamaKost, p.Nama, p.NoKamar, p.CheckIn, strings.Title(p.TipeHarga), p.Harga,
	)

	f, err := os.Open(pdfPath)
	if err != nil {
		return err
	}
	defer f.Close()

	body := &strings.Builder{}
	boundary := "----FormBoundary7MA4YWxkTrZu0gW"

	// Write caption field
	body.WriteString("--" + boundary + "\r\n")
	body.WriteString("Content-Disposition: form-data; name=\"chat_id\"\r\n\r\n")
	body.WriteString(s.TelegramChatID + "\r\n")
	body.WriteString("--" + boundary + "\r\n")
	body.WriteString("Content-Disposition: form-data; name=\"caption\"\r\n\r\n")
	body.WriteString(caption + "\r\n")
	body.WriteString("--" + boundary + "\r\n")
	body.WriteString("Content-Disposition: form-data; name=\"parse_mode\"\r\n\r\nMarkdown\r\n")

	// Use python to send (simpler for multipart)
	script := fmt.Sprintf(`
import urllib.request, urllib.parse, json, os

token = %q
chat_id = %q
caption = %q
pdf_path = %q

with open(pdf_path, 'rb') as f:
    pdf_data = f.read()

boundary = b'----Boundary7MA4YWxk'

def make_field(name, value):
    return (b'--' + boundary + b'\r\n' +
            b'Content-Disposition: form-data; name="' + name.encode() + b'"\r\n\r\n' +
            value.encode() + b'\r\n')

def make_file(name, filename, data):
    return (b'--' + boundary + b'\r\n' +
            b'Content-Disposition: form-data; name="' + name.encode() + b'"; filename="' + filename.encode() + b'"\r\n' +
            b'Content-Type: application/pdf\r\n\r\n' +
            data + b'\r\n')

body = (make_field('chat_id', chat_id) +
        make_field('caption', caption) +
        make_field('parse_mode', 'Markdown') +
        make_file('document', os.path.basename(pdf_path), pdf_data) +
        b'--' + boundary + b'--\r\n')

req = urllib.request.Request(
    f'https://api.telegram.org/bot{token}/sendDocument',
    data=body,
    headers={'Content-Type': 'multipart/form-data; boundary=' + boundary.decode()}
)
try:
    resp = urllib.request.urlopen(req, timeout=15)
    print('ok')
except Exception as e:
    print('error:', e)
`, s.TelegramToken, s.TelegramChatID, caption, pdfPath)

	cmd := exec.Command("python3", "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("telegram send failed: %v | %s", err, out)
	}
	if strings.Contains(string(out), "error:") {
		return fmt.Errorf("telegram: %s", out)
	}
	log.Printf("Telegram sent: %s", out)
	return nil
}

// ─────────────────────────────────────────────
// HTML Template (SPA)
// ─────────────────────────────────────────────



// ─────────────────────────────────────────────
// Main
// ─────────────────────────────────────────────

func main() {
	initDB()

	mux := http.NewServeMux()

	// Serve index
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "web/index.html")
	})

	// API routes
	mux.HandleFunc("/api/settings", handleSettings)
	mux.HandleFunc("/api/penghuni", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			handleListPenghuni(w, r)
		case "POST":
			handleAddPenghuni(w, r)
		default:
			http.Error(w, "method not allowed", 405)
		}
	})
	mux.HandleFunc("/api/penghuni/", handleDeletePenghuni)
	mux.HandleFunc("/api/invoice/", handleInvoice)
	mux.HandleFunc("/api/resend/", handleResend)

	// CORS middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		mux.ServeHTTP(w, r)
	})

	port := ":8080"
	log.Printf("🏠 Kost App running at http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, handler))
}
