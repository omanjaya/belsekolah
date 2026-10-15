package store

import (
	"errors"
	"testing"
	"time"

	"belsekolah/internal/formatjadwal"
)

func TestTemplateCRUD(t *testing.T) {
	s := bukaStore(t)
	d := dataAsing()
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	id := harus(s.CreateTemplate(ctx, "Template A", d, now))(t)
	if _, err := s.CreateTemplate(ctx, "Template A", d, now); !errors.Is(err, ErrDuplikat) {
		t.Fatalf("nama ganda: err = %v", err)
	}
	bad := d
	bad.Jadwal = []formatjadwal.Bel{{Hari: 0, Jam: "07:00", JenisBel: "MASUK"}}
	_, err := s.CreateTemplate(ctx, "Rusak", bad, now)
	harapValidasi(t, err)

	tp := harus(s.GetTemplate(ctx, id))(t)
	if tp.Nama != "Template A" || tp.Data.Nama != "Template A" || len(tp.Data.Jadwal) != 3 || !tp.Dibuat.Equal(now) {
		t.Fatalf("template = %+v", tp)
	}
	if err := s.UpdateNamaTemplate(ctx, id, "Template B"); err != nil {
		t.Fatal(err)
	}
	tp = harus(s.GetTemplate(ctx, id))(t)
	if tp.Nama != "Template B" || tp.Data.Nama != "Template B" {
		t.Fatalf("ganti nama: %+v", tp)
	}
	harus(s.CreateTemplate(ctx, "Awal", d, now))(t)
	list := harus(s.ListTemplate(ctx))(t)
	if len(list) != 2 || list[0].Nama != "Awal" {
		t.Fatalf("list harus urut nama: %+v", list)
	}
	if err := s.DeleteTemplate(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetTemplate(ctx, id); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("setelah hapus: %v", err)
	}
	if err := s.DeleteTemplate(ctx, id); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("hapus dua kali: %v", err)
	}
}
