package zona

import "testing"

func TestIANAdanKode(t *testing.T) {
	for _, c := range []struct{ kode, iana string }{
		{"WIB", "Asia/Jakarta"}, {"WITA", "Asia/Makassar"}, {"WIT", "Asia/Jayapura"},
	} {
		got, ok := IANA(c.kode)
		if !ok || got != c.iana {
			t.Errorf("IANA(%q) = %q,%v; mau %q", c.kode, got, ok, c.iana)
		}
		if k := Kode(c.iana); k != c.kode {
			t.Errorf("Kode(%q) = %q; mau %q", c.iana, k, c.kode)
		}
	}
	if _, ok := IANA("wita"); ok {
		t.Error("kode harus huruf besar persis")
	}
	if k := Kode("Europe/Berlin"); k != "Europe/Berlin" {
		t.Errorf("Kode di luar daftar = %q; mau nama IANA apa adanya", k)
	}
}

func TestPilih(t *testing.T) {
	cases := []struct {
		nama, tersimpan, flag, mau string
		peringatan                 bool
	}{
		{"tersimpan menang", "Asia/Makassar", "Asia/Jakarta", "Asia/Makassar", false},
		{"kosong pakai flag", "", "Asia/Jayapura", "Asia/Jayapura", false},
		{"tersimpan rusak pakai flag", "Bukan/Zona", "Asia/Jakarta", "Asia/Jakarta", true},
		{"flag rusak pakai Jakarta", "", "Bukan/Zona", "Asia/Jakarta", true},
		{"semua kosong", "", "", "Asia/Jakarta", false},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			loc, per := Pilih(c.tersimpan, c.flag)
			if loc.String() != c.mau {
				t.Errorf("zona = %q; mau %q", loc.String(), c.mau)
			}
			if (len(per) > 0) != c.peringatan {
				t.Errorf("peringatan = %v; mau ada=%v", per, c.peringatan)
			}
		})
	}
}
