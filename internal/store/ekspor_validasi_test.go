package store

import (
	"fmt"
	"strings"
	"testing"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/schedule"
)

func TestEksporProfil_TerlaluBanyakBel(t *testing.T) {
	s := bukaStore(t)
	profil, jenis, _ := dataDasar(t, s)
	n := 0
	for hari := 1; hari <= 7 && n <= formatjadwal.MaksBel; hari++ {
		for menit := 0; menit < 1440 && n <= formatjadwal.MaksBel; menit++ {
			jam := fmt.Sprintf("%02d:%02d", menit/60, menit%60)
			harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: hari, Jam: jam, JenisBelID: jenis}))(t)
			n++
		}
	}
	_, err := s.EksporProfil(ctx, profil)
	harapValidasi(t, err)
	if !strings.Contains(err.Error(), "tidak bisa diekspor") {
		t.Fatalf("pesan = %q", err.Error())
	}
}

func TestEksporProfil_JenisBedaHurufBesarKecil(t *testing.T) {
	s := bukaStore(t)
	profil, jenis, _ := dataDasar(t, s)
	jenis2, err := s.CreateJenisBel(ctx, "MASUK", "#111111")
	if err != nil {
		t.Skipf("store menolak jenis beda huruf besar/kecil: %v", err)
	}
	harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "07:00", JenisBelID: jenis}))(t)
	harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "08:00", JenisBelID: jenis2}))(t)
	_, err = s.EksporProfil(ctx, profil)
	harapValidasi(t, err)
	if !strings.Contains(err.Error(), "hanya beda huruf besar/kecil") {
		t.Fatalf("pesan = %q", err.Error())
	}
}
