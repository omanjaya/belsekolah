package store

import (
	"errors"
	"fmt"
	"testing"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/gabung"
	"belsekolah/internal/schedule"
)

// dataGabung: Sen 07:00 Masuk "Upacara" (bentrok), Sen 12:00 Masuk "Pulang" (kembar),
// Sel 07:00 Masuk "Masuk" (baru), Jum 07:00 masuk "" (kembar: nama jenis beda huruf).
func dataGabung() formatjadwal.Data {
	d := formatjadwal.Baru("Tambah banyak")
	d.JenisBel = []formatjadwal.JenisBel{{Nama: "Masuk", Warna: "#2563eb"}}
	d.Jadwal = []formatjadwal.Bel{
		{Hari: 1, Jam: "07:00", JenisBel: "Masuk", Label: "Upacara"},
		{Hari: 1, Jam: "12:00", JenisBel: "Masuk", Label: "Pulang"},
		{Hari: 2, Jam: "07:00", JenisBel: "Masuk", Label: "Masuk"},
		{Hari: 5, Jam: "07:00", JenisBel: "MASUK", Label: ""},
	}
	return d
}

func TestPeriksaGabung(t *testing.T) {
	s := bukaStore(t)
	profil, _ := isiProfil(t, s)
	h := harus(s.PeriksaGabung(ctx, profil, dataGabung(), nil))(t)
	if h.Ringkasan != (RingkasanGabung{Baru: 1, Kembar: 2, Bentrok: 1}) {
		t.Fatalf("ringkasan = %+v", h.Ringkasan)
	}
	if len(h.JenisBelTakDikenal) != 0 {
		t.Fatalf("tak dikenal = %v", h.JenisBelTakDikenal)
	}
	b := h.Baris[0]
	if b.Hari != 1 || b.Jam != "07:00" || b.Status != gabung.Bentrok || b.Lama == nil ||
		b.Lama.JenisBel != "Masuk" || b.Lama.Label != "Masuk" || b.Baru.Label != "Upacara" {
		t.Fatalf("baris[0] = %+v lama=%+v", b, b.Lama)
	}
	if h.Baris[2].Status != gabung.Baru || h.Baris[2].Lama != nil {
		t.Fatalf("baris[2] = %+v", h.Baris[2])
	}
	// Periksa tidak mengubah apa pun.
	if n := len(harus(s.ListJadwal(ctx, profil))(t)); n != 3 {
		t.Fatalf("jadwal = %d setelah periksa", n)
	}
}

func TestPeriksaGabung_JenisTakDikenal(t *testing.T) {
	s := bukaStore(t)
	profil, jenis := isiProfil(t, s)
	d := dataAsing() // Sen 07:00 MASUK "" (lama: Masuk "Masuk") → bentrok karena label beda
	h := harus(s.PeriksaGabung(ctx, profil, d, nil))(t)
	if fmt.Sprint(h.JenisBelTakDikenal) != "[Bel Literasi Bel Doa]" {
		t.Fatalf("tak dikenal = %v", h.JenisBelTakDikenal)
	}
	if h.Ringkasan.Baru != 2 || h.Ringkasan.Bentrok != 1 {
		t.Fatalf("ringkasan = %+v", h.Ringkasan)
	}
	// Setelah dipetakan, daftar tak dikenal tetap dilaporkan.
	h = harus(s.PeriksaGabung(ctx, profil, d, map[string]PemetaanJenis{
		"Bel Literasi": {Baru: true}, "Bel Doa": {JenisBelID: jenis},
	}))(t)
	if len(h.JenisBelTakDikenal) != 2 {
		t.Fatalf("tak dikenal setelah pemetaan = %v", h.JenisBelTakDikenal)
	}
}

func TestPeriksaGabung_ProfilTidakAda(t *testing.T) {
	s := bukaStore(t)
	isiProfil(t, s)
	if _, err := s.PeriksaGabung(ctx, 999, dataGabung(), nil); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("err = %v", err)
	}
	if _, err := s.GabungJadwal(ctx, 999, dataGabung(), nil, nil); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("err = %v", err)
	}
}

func TestGabungJadwal(t *testing.T) {
	s := bukaStore(t)
	profil, jenis := isiProfil(t, s)
	h := harus(s.GabungJadwal(ctx, profil, dataGabung(), nil, []gabung.Slot{{Hari: 1, Jam: "07:00"}, {Hari: 1, Jam: "12:00"}}))(t)
	if h != (HasilGabung{Ditambah: 1, Ditimpa: 1, Dilewati: 2}) {
		t.Fatalf("hasil = %+v", h)
	}
	js := harus(s.ListJadwal(ctx, profil))(t)
	got := map[string]string{}
	for _, j := range js {
		if j.JenisBelID != jenis {
			t.Fatalf("jenis salah: %+v", j)
		}
		got[fmt.Sprintf("%d %s", j.Hari, j.Jam)] = j.Label
	}
	mau := map[string]string{"1 07:00": "Upacara", "1 12:00": "Pulang", "2 07:00": "Masuk", "5 07:00": ""}
	if fmt.Sprint(got) != fmt.Sprint(mau) {
		t.Fatalf("jadwal = %v, mau %v", got, mau)
	}
}

func TestGabungJadwal_PertahankanBawaan(t *testing.T) {
	s := bukaStore(t)
	profil, _ := isiProfil(t, s)
	h := harus(s.GabungJadwal(ctx, profil, dataGabung(), nil, nil))(t)
	if h != (HasilGabung{Ditambah: 1, Ditimpa: 0, Dilewati: 3}) {
		t.Fatalf("hasil = %+v", h)
	}
	for _, j := range harus(s.ListJadwal(ctx, profil))(t) {
		if j.Hari == 1 && j.Jam == "07:00" && j.Label != "Masuk" {
			t.Fatalf("bel lama berubah: %+v", j)
		}
	}
}

func TestGabungJadwal_JenisBaruDanPemetaan(t *testing.T) {
	s := bukaStore(t)
	profil, jenis := isiProfil(t, s)
	d := dataAsing()
	if _, err := s.GabungJadwal(ctx, profil, d, map[string]PemetaanJenis{"Bel Literasi": {Baru: true}}, nil); err == nil {
		t.Fatal("pemetaan kurang harus ditolak")
	} else {
		harapValidasi(t, err)
	}
	if n := len(harus(s.ListJenisBel(ctx))(t)); n != 1 {
		t.Fatalf("jenis bel = %d setelah gagal (transaksi harus batal)", n)
	}
	h := harus(s.GabungJadwal(ctx, profil, d, map[string]PemetaanJenis{
		"Bel Literasi": {Baru: true}, "Bel Doa": {JenisBelID: jenis},
	}, nil))(t)
	if h.Ditambah != 2 {
		t.Fatalf("hasil = %+v", h)
	}
	semua := harus(s.ListJenisBel(ctx))(t)
	if len(semua) != 2 || semua[1].Nama != "Bel Literasi" {
		t.Fatalf("jenis = %+v", semua)
	}
	if n := len(harus(s.UrutanBel(ctx, semua[1].ID))(t)); n != 1 {
		t.Fatalf("jenis baru harus punya 1 suara, ada %d", n)
	}
}

func TestGabungJadwal_BatasTotal(t *testing.T) {
	s := bukaStore(t)
	profil, jenis := isiProfil(t, s) // 3 bel
	// Isi profil sampai 999 bel: hari 2–7 × jam 00:00.. (langsung lewat CreateJadwal).
	n := 3
	for h := 2; h <= 7 && n < 999; h++ {
		for m := 0; m < 24*60 && n < 999; m += 1 {
			jam := fmt.Sprintf("%02d:%02d", m/60, m%60)
			harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: h, Jam: jam, JenisBelID: jenis}))(t)
			n++
		}
	}
	d := formatjadwal.Baru("Tambah")
	d.JenisBel = []formatjadwal.JenisBel{{Nama: "Masuk", Warna: "#2563eb"}}
	d.Jadwal = []formatjadwal.Bel{
		{Hari: 1, Jam: "08:00", JenisBel: "Masuk"},
		{Hari: 1, Jam: "08:01", JenisBel: "Masuk"},
	}
	_, err := s.GabungJadwal(ctx, profil, d, nil, nil)
	harapValidasi(t, err)
	if c := len(harus(s.ListJadwal(ctx, profil))(t)); c != 999 {
		t.Fatalf("jadwal = %d setelah gagal", c)
	}
	d.Jadwal = d.Jadwal[:1]
	harus(s.GabungJadwal(ctx, profil, d, nil, nil))(t) // tepat 1000: boleh
}

func TestGabungJadwal_DataTidakValid(t *testing.T) {
	s := bukaStore(t)
	profil, _ := isiProfil(t, s)
	d := dataGabung()
	d.Jadwal[0].Jam = "7:00"
	_, err := s.GabungJadwal(ctx, profil, d, nil, nil)
	harapValidasi(t, err)
	_, err = s.PeriksaGabung(ctx, profil, d, nil)
	harapValidasi(t, err)
}
