package store

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	ErrDipakai  = errors.New("data masih dipakai")
	ErrDuplikat = errors.New("data dengan nama atau waktu yang sama sudah ada")
)

// ErrorValidasi menandai input yang tidak valid; Pesan aman ditampilkan ke pengguna.
type ErrorValidasi struct {
	Pesan string
}

func (e *ErrorValidasi) Error() string { return e.Pesan }

func validasi(format string, a ...any) error {
	return &ErrorValidasi{Pesan: fmt.Sprintf(format, a...)}
}

// dipakai membuat error ErrDipakai beserta alasannya.
func dipakai(alasan string) error {
	return fmt.Errorf("%w: %s", ErrDipakai, alasan)
}

// terjemahkan mengubah error constraint SQLite menjadi error domain.
// errFK dikembalikan untuk pelanggaran foreign key; nil berarti kembalikan err apa adanya.
func terjemahkan(err, errFK error) error {
	if err == nil {
		return nil
	}
	pesan := err.Error()
	switch {
	case strings.Contains(pesan, "UNIQUE constraint failed"):
		return ErrDuplikat
	case strings.Contains(pesan, "FOREIGN KEY constraint failed"):
		if errFK == nil {
			return err
		}
		return errFK
	}
	return err
}

// cekDiubah mengembalikan ErrTidakAda jika perintah tidak mengenai baris mana pun.
func cekDiubah(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTidakAda
	}
	return nil
}

func validasiTeks(nilai, nama string, maks int, wajib bool) (string, error) {
	nilai = strings.TrimSpace(nilai)
	if wajib && nilai == "" {
		return "", validasi("%s tidak boleh kosong", nama)
	}
	if utf8.RuneCountInString(nilai) > maks {
		return "", validasi("%s maksimal %d karakter", nama, maks)
	}
	return nilai, nil
}

func validasiNama(nama string) (string, error) {
	return validasiTeks(nama, "nama", 100, true)
}

var polaWarna = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func validasiWarna(w string) error {
	if !polaWarna.MatchString(w) {
		return validasi("warna harus berformat #RRGGBB")
	}
	return nil
}
