// Package bawaan memasang suara bawaan dan data awal untuk instalasi baru.
package bawaan

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"belsekolah/internal/nada"
	"belsekolah/internal/store"
)

type jenisAwal struct {
	nama  string
	warna string
	audio []string // nama audio bawaan, diputar berurutan
}

var jenisBelAwal = []jenisAwal{
	{"Masuk", "#2563eb", []string{"Ding Dong Panjang"}},
	{"Ganti Pelajaran", "#0d9488", []string{"Ding Dong"}},
	{"Istirahat", "#16a34a", []string{"Ding Dong", "Ding Dong"}},
	{"Pulang", "#9333ea", []string{"Ding Dong Panjang", "Ding Dong Panjang"}},
	{"Darurat", "#dc2626", []string{"Sirene", "Sirene"}},
}

// Pasang menulis file suara bawaan yang belum ada, mendaftarkannya ke database,
// dan—hanya jika belum ada profil—membuat profil "Normal" serta jenis bel awal.
func Pasang(ctx context.Context, st *store.Store, audioDir string, now time.Time) error {
	ids := make(map[string]int64)
	for _, b := range nada.Daftar {
		data := b.Data()
		path := filepath.Join(audioDir, b.File)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(path, data, 0o644); err != nil {
				return fmt.Errorf("tulis %s: %w", b.File, err)
			}
		}
		a, err := st.FindAudio(ctx, "bawaan", b.Nama)
		switch {
		case errors.Is(err, store.ErrTidakAda):
			id, err := st.CreateAudio(ctx, store.Audio{
				Nama: b.Nama, Sumber: "bawaan", File: b.File,
				DurasiMS: nada.DurasiMS(data), Dibuat: now,
			})
			if err != nil {
				return fmt.Errorf("daftarkan %s: %w", b.Nama, err)
			}
			ids[b.Nama] = id
		case err != nil:
			return err
		default:
			ids[b.Nama] = a.ID
		}
	}

	n, err := st.CountProfil(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	for _, j := range jenisBelAwal {
		id, err := st.CreateJenisBel(ctx, j.nama, j.warna)
		if err != nil {
			return fmt.Errorf("buat jenis bel %s: %w", j.nama, err)
		}
		var audioIDs []int64
		for _, nama := range j.audio {
			audioIDs = append(audioIDs, ids[nama])
		}
		if err := st.SetUrutanBel(ctx, id, audioIDs); err != nil {
			return fmt.Errorf("atur suara %s: %w", j.nama, err)
		}
	}
	// Profil dibuat terakhir: keberadaan profil menandai data awal sudah lengkap.
	profil, err := st.CreateProfil(ctx, "Normal")
	if err != nil {
		return err
	}
	return st.SetSetting(ctx, store.KunciProfilDefault, strconv.FormatInt(profil, 10))
}
