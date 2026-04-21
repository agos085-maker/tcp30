#!/usr/bin/env python3
"""
Invoice PDF Generator untuk Kost App
Dipanggil oleh Go backend: python3 gen_invoice.py '<json>'
"""
import sys, json, os
from fpdf import FPDF
from datetime import datetime

def format_rp(amount):
    """Format angka ke format Rupiah"""
    return "Rp {:,.0f}".format(float(amount)).replace(",", ".")

def generate_invoice(data):
    pdf = FPDF()
    pdf.add_page()
    pdf.set_auto_page_break(auto=True, margin=15)

    # ── Color palette ──
    DARK      = (15, 17, 23)
    ACCENT    = (108, 99, 255)
    ACCENT2   = (67, 233, 123)
    LIGHT_BG  = (248, 249, 252)
    MUTED     = (120, 128, 160)
    WHITE     = (255, 255, 255)
    BORDER    = (230, 232, 240)

    # ── Header Background ──
    pdf.set_fill_color(*DARK)
    pdf.rect(0, 0, 210, 55, 'F')

    # Accent strip
    pdf.set_fill_color(*ACCENT)
    pdf.rect(0, 55, 210, 3, 'F')

    # ── Logo / Nama Kost ──
    pdf.set_text_color(*WHITE)
    pdf.set_font("Helvetica", "B", 22)
    pdf.set_xy(14, 12)
    pdf.cell(0, 10, data.get("nama_kost", "Kost"), ln=True)

    pdf.set_font("Helvetica", "", 10)
    pdf.set_text_color(*MUTED)
    pdf.set_xy(14, 23)
    pdf.cell(0, 6, data.get("alamat", ""), ln=True)

    # ── INVOICE label (top right) ──
    pdf.set_text_color(*ACCENT)
    pdf.set_font("Helvetica", "B", 28)
    pdf.set_xy(110, 10)
    pdf.cell(85, 12, "INVOICE", align="R")

    pdf.set_text_color(180, 185, 210)
    pdf.set_font("Helvetica", "", 10)
    pdf.set_xy(110, 24)
    pdf.cell(85, 6, f"No. INV-{str(data.get('id', 0)).zfill(5)}", align="R")
    pdf.set_xy(110, 31)
    pdf.cell(85, 6, f"Tanggal: {data.get('tanggal', '')}", align="R")

    # ── Info Box ──
    y_info = 65
    # Left - Penghuni
    pdf.set_fill_color(*LIGHT_BG)
    pdf.set_draw_color(*BORDER)
    pdf.rect(14, y_info, 85, 48, 'FD')

    pdf.set_text_color(*MUTED)
    pdf.set_font("Helvetica", "B", 8)
    pdf.set_xy(19, y_info + 5)
    pdf.cell(0, 5, "PENGHUNI")

    pdf.set_text_color(*DARK)
    pdf.set_font("Helvetica", "B", 13)
    pdf.set_xy(19, y_info + 12)
    pdf.cell(0, 7, data.get("nama", ""))

    pdf.set_font("Helvetica", "", 10)
    pdf.set_text_color(*MUTED)
    pdf.set_xy(19, y_info + 21)
    pdf.cell(0, 6, f"No. HP: {data.get('no_hp', '-') or '-'}")
    pdf.set_xy(19, y_info + 29)
    pdf.cell(0, 6, f"Check-in: {data.get('check_in', '')}")

    # Right - Kamar
    pdf.set_fill_color(*ACCENT)
    pdf.rect(111, y_info, 85, 48, 'F')

    pdf.set_text_color(*WHITE)
    pdf.set_font("Helvetica", "B", 8)
    pdf.set_xy(116, y_info + 5)
    pdf.cell(0, 5, "NOMOR KAMAR")

    pdf.set_font("Helvetica", "B", 36)
    pdf.set_xy(116, y_info + 12)
    pdf.cell(74, 18, data.get("no_kamar", ""), align="C")

    tipe = data.get("tipe_harga", "bulanan").upper()
    pdf.set_font("Helvetica", "", 10)
    pdf.set_text_color(200, 198, 255)
    pdf.set_xy(116, y_info + 33)
    pdf.cell(74, 6, f"Tarif {tipe}", align="C")

    # ── Invoice Table ──
    y_table = y_info + 60
    pdf.set_fill_color(*DARK)
    pdf.rect(14, y_table, 182, 10, 'F')

    pdf.set_text_color(*WHITE)
    pdf.set_font("Helvetica", "B", 9)
    headers = ["Deskripsi", "Tipe", "Kamar", "Harga"]
    widths   = [80, 35, 35, 32]
    x = 14
    for h, w in zip(headers, widths):
        pdf.set_xy(x + 3, y_table + 2)
        pdf.cell(w, 6, h)
        x += w

    y_row = y_table + 10
    pdf.set_fill_color(*LIGHT_BG)
    pdf.rect(14, y_row, 182, 12, 'F')

    pdf.set_text_color(*DARK)
    pdf.set_font("Helvetica", "", 10)
    desc = f"Sewa Kamar - {data.get('nama_kost','')}"
    row_data = [
        desc,
        data.get("tipe_harga", "bulanan").title(),
        data.get("no_kamar", ""),
        format_rp(data.get("harga", 0))
    ]
    x = 14
    for val, w in zip(row_data, widths):
        pdf.set_xy(x + 3, y_row + 3)
        pdf.cell(w, 6, str(val))
        x += w

    # ── Total Box ──
    y_total = y_row + 22
    pdf.set_fill_color(*ACCENT)
    pdf.rect(110, y_total, 86, 22, 'F')

    pdf.set_text_color(*WHITE)
    pdf.set_font("Helvetica", "", 10)
    pdf.set_xy(115, y_total + 4)
    pdf.cell(40, 7, "TOTAL TAGIHAN")

    pdf.set_font("Helvetica", "B", 15)
    pdf.set_xy(115, y_total + 11)
    pdf.cell(76, 8, format_rp(data.get("harga", 0)), align="R")

    # ── Catatan ──
    y_note = y_total + 32
    pdf.set_text_color(*MUTED)
    pdf.set_font("Helvetica", "I", 9)
    pdf.set_xy(14, y_note)
    pdf.multi_cell(182, 5, "Terima kasih telah mempercayai kami. Harap melakukan pembayaran sesuai jadwal yang telah disepakati.", align="C")

    # ── Footer ──
    pdf.set_fill_color(*DARK)
    pdf.rect(0, 277, 210, 20, 'F')
    pdf.set_text_color(*MUTED)
    pdf.set_font("Helvetica", "", 8)
    pdf.set_xy(0, 282)
    pdf.cell(210, 5, f"{data.get('nama_kost','')}  |  {data.get('alamat','')}  |  Dicetak: {data.get('tanggal','')}", align="C")

    # ── Save ──
    out_path = data.get("pdf_path", f"invoices/invoice_{data.get('id',0)}.pdf")
    os.makedirs(os.path.dirname(out_path), exist_ok=True)
    pdf.output(out_path)
    print(f"PDF generated: {out_path}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 gen_invoice.py '<json_data>'")
        sys.exit(1)
    try:
        data = json.loads(sys.argv[1])
        generate_invoice(data)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
