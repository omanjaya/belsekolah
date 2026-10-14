// Package zona mengenal zona waktu Indonesia (WIB/WITA/WIT) dan memilih zona aktif saat start.
package zona

import (
	"fmt"
	"time"
	_ "time/tzdata" // zona tertanam, agar tidak bergantung pada tzdata OS
)

// Bawaan dipakai bila tidak ada zona lain yang valid.
const Bawaan = "Asia/Jakarta"

var daftar = []struct{ kode, iana string }{
	{"WIB", "Asia/Jakarta"},
	{"WITA", "Asia/Makassar"},
	{"WIT", "Asia/Jayapura"},
}

// IANA mengubah WIB/WITA/WIT menjadi nama zona IANA.
func IANA(kode string) (string, bool) {
	for _, z := range daftar {
		if z.kode == kode {
			return z.iana, true
		}
	}
	return "", false
}

// Kode mengubah nama IANA menjadi WIB/WITA/WIT; zona lain dikembalikan apa adanya.
func Kode(iana string) string {
	for _, z := range daftar {
		if z.iana == iana {
			return z.kode
		}
	}
	return iana
}

// Pilih menentukan zona aktif: zona tersimpan bila valid, lalu flag, lalu Asia/Jakarta.
// Nilai yang tidak valid dilewati dan dilaporkan sebagai peringatan.
func Pilih(tersimpan, flag string) (*time.Location, []string) {
	var peringatan []string
	for _, c := range []struct{ asal, nama string }{{"pengaturan", tersimpan}, {"flag -tz", flag}} {
		if c.nama == "" {
			continue
		}
		loc, err := time.LoadLocation(c.nama)
		if err == nil {
			return loc, peringatan
		}
		peringatan = append(peringatan, fmt.Sprintf("zona waktu dari %s tidak valid (%q), diabaikan", c.asal, c.nama))
	}
	loc, _ := time.LoadLocation(Bawaan)
	return loc, peringatan
}
