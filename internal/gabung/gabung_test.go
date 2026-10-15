package gabung

import (
	"reflect"
	"testing"
)

func TestKlasifikasi(t *testing.T) {
	lama := []Lama{
		{JadwalID: 10, Hari: 1, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "Masuk"}},
		{JadwalID: 11, Hari: 1, Jam: "09:00", Isi: Isi{JenisBelID: 2, Label: "Istirahat"}},
		{JadwalID: 12, Hari: 2, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "Masuk"}},
		{JadwalID: 13, Hari: 3, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "Masuk"}},
		{JadwalID: 14, Hari: 5, Jam: "13:00", Isi: Isi{JenisBelID: 1, Label: "Pulang"}}, // tidak disentuh
	}
	masuk := []Masuk{
		{Hari: 3, Jam: "07:00", Isi: Isi{JenisBelID: 0, Label: "Masuk"}},       // 0: jenis belum dipetakan → bentrok
		{Hari: 1, Jam: "09:00", Isi: Isi{JenisBelID: 2, Label: " Istirahat "}}, // label di-trim → kembar
		{Hari: 1, Jam: "07:15", Isi: Isi{JenisBelID: 1, Label: "Dhuha"}},       // baru
		{Hari: 1, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "Upacara"}},     // beda label → bentrok
		{Hari: 2, Jam: "07:00", Isi: Isi{JenisBelID: 3, Label: "Masuk"}},       // beda jenis → bentrok
		{Hari: 2, Jam: "07:01", Isi: Isi{JenisBelID: 1, Label: "masuk"}},       // baru
	}
	got := Klasifikasi(lama, masuk)
	type ringkas struct {
		Indeks   int
		Hari     int
		Jam      string
		Status   Status
		JadwalID int64
	}
	var r []ringkas
	for _, b := range got {
		var id int64
		if b.Lama != nil {
			id = b.Lama.JadwalID
		}
		r = append(r, ringkas{b.Indeks, b.Hari, b.Jam, b.Status, id})
	}
	mau := []ringkas{
		{3, 1, "07:00", Bentrok, 10},
		{2, 1, "07:15", Baru, 0},
		{1, 1, "09:00", Kembar, 11},
		{4, 2, "07:00", Bentrok, 12},
		{5, 2, "07:01", Baru, 0},
		{0, 3, "07:00", Bentrok, 13},
	}
	if !reflect.DeepEqual(r, mau) {
		t.Fatalf("Klasifikasi =\n%+v\nmau\n%+v", r, mau)
	}
	if got[0].Baru.Label != "Upacara" || got[0].Lama.Label != "Masuk" {
		t.Fatalf("isi baris pertama salah: %+v", got[0])
	}
}

func TestKlasifikasi_LabelPekaHuruf(t *testing.T) {
	lama := []Lama{{JadwalID: 1, Hari: 1, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "Upacara"}}}
	got := Klasifikasi(lama, []Masuk{{Hari: 1, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "upacara"}}})
	if got[0].Status != Bentrok {
		t.Fatalf("status = %s, mau bentrok", got[0].Status)
	}
}

func TestRencanakan(t *testing.T) {
	baris := Klasifikasi(
		[]Lama{
			{JadwalID: 10, Hari: 1, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "A"}},
			{JadwalID: 11, Hari: 1, Jam: "08:00", Isi: Isi{JenisBelID: 1, Label: "B"}},
			{JadwalID: 12, Hari: 1, Jam: "09:00", Isi: Isi{JenisBelID: 1, Label: "C"}},
		},
		[]Masuk{
			{Hari: 1, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "A2"}}, // bentrok, ditimpa
			{Hari: 1, Jam: "08:00", Isi: Isi{JenisBelID: 1, Label: "B2"}}, // bentrok, dipertahankan
			{Hari: 1, Jam: "09:00", Isi: Isi{JenisBelID: 1, Label: "C"}},  // kembar
			{Hari: 2, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "D"}},  // baru
		},
	)
	r := Rencanakan(baris, []Slot{
		{Hari: 1, Jam: "07:00"},
		{Hari: 1, Jam: "09:00"}, // kembar: diabaikan
		{Hari: 2, Jam: "07:00"}, // baru: tetap ditambah, bukan ditimpa
		{Hari: 4, Jam: "10:00"}, // tidak ada di masukan: diabaikan
	})
	if len(r.Tambah) != 1 || r.Tambah[0].Hari != 2 {
		t.Fatalf("Tambah = %+v", r.Tambah)
	}
	if len(r.Timpa) != 1 || r.Timpa[0].Lama.JadwalID != 10 || r.Timpa[0].Baru.Label != "A2" {
		t.Fatalf("Timpa = %+v", r.Timpa)
	}
	if r.Dilewati != 2 {
		t.Fatalf("Dilewati = %d, mau 2", r.Dilewati)
	}
}

func TestKlasifikasi_Kosong(t *testing.T) {
	if got := Klasifikasi(nil, nil); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
	r := Rencanakan(nil, nil)
	if r.Tambah == nil || r.Timpa == nil {
		t.Fatal("slice Rencana harus non-nil")
	}
}
