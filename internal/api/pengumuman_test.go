package api

import (
	"strings"
	"testing"
	"time"

	"belsekolah/internal/schedule"
	"belsekolah/internal/scheduler"
	"belsekolah/internal/store"
)

func suaraUji(t *testing.T, l *lingkungan, nama string) int64 {
	t.Helper()
	id, err := l.st.CreateAudio(ctx, store.Audio{Nama: nama, Sumber: "upload", File: nama + ".wav", DurasiMS: 1000})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

type jadwalResp struct {
	ID           int64  `json:"id"`
	PengumumanID *int64 `json:"pengumumanId"`
}

func TestPengumuman_JadwalCRUD(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	profil, _ := l.st.ProfilDefault(ctx)
	js, _ := l.st.ListJenisBel(ctx)
	au := suaraUji(t, l, "Jam ke-3 dimulai")
	base := map[string]any{"profilId": profil, "hari": 1, "jam": "08:00", "jenisBelId": js[0].ID, "label": "Jam ke-3"}

	// tanpa field -> null
	resp := kirim(t, c, "POST", l.srv.URL+"/api/jadwal", base)
	harapKode(t, resp, 201)
	var j jadwalResp
	bacaBody(t, resp, &j)
	if j.PengumumanID != nil {
		t.Fatalf("harus null, dapat %v", *j.PengumumanID)
	}

	// ubah dengan pengumumanId
	base["pengumumanId"] = au
	resp = kirim(t, c, "PUT", l.srv.URL+"/api/jadwal/"+itoa(j.ID), base)
	harapKode(t, resp, 200)
	var j2 jadwalResp
	bacaBody(t, resp, &j2)
	if j2.PengumumanID == nil || *j2.PengumumanID != au {
		t.Fatalf("pengumumanId = %v", j2.PengumumanID)
	}

	// kembali null
	base["pengumumanId"] = nil
	resp = kirim(t, c, "PUT", l.srv.URL+"/api/jadwal/"+itoa(j.ID), base)
	harapKode(t, resp, 200)
	var j3 jadwalResp
	bacaBody(t, resp, &j3)
	if j3.PengumumanID != nil {
		t.Fatal("harus null kembali")
	}

	// tidak ada -> 400
	base["pengumumanId"] = 99999
	harapKode(t, kirim(t, c, "POST", l.srv.URL+"/api/jadwal", base), 400)
}

func TestPengumuman_Cocok(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	au := suaraUji(t, l, "Jam ke-3 dimulai")
	var o struct {
		Suara *store.SuaraRingkas `json:"suara"`
	}
	resp := kirim(t, c, "GET", l.srv.URL+"/api/pengumuman/cocok?keterangan=Jam+ke-3", nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &o)
	if o.Suara == nil || o.Suara.ID != au {
		t.Fatalf("suara = %+v", o.Suara)
	}
	o.Suara = &store.SuaraRingkas{}
	resp = kirim(t, c, "GET", l.srv.URL+"/api/pengumuman/cocok?keterangan=zzz+tidak+ada", nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &o)
	if o.Suara != nil {
		t.Fatalf("harus null: %+v", o.Suara)
	}
}

func TestPengumuman_SaranTerapkan(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	profil, _ := l.st.ProfilDefault(ctx)
	js, _ := l.st.ListJenisBel(ctx)
	au := suaraUji(t, l, "Jam ke-3 dimulai")
	if _, err := l.st.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "08:00", JenisBelID: js[0].ID, Label: "Jam ke-3"}); err != nil {
		t.Fatal(err)
	}
	url := l.srv.URL + "/api/profil/" + itoa(profil) + "/pengumuman/"
	var saran []store.SaranPengumuman
	resp := kirim(t, c, "GET", url+"saran", nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &saran)
	if len(saran) != 1 || saran[0].Suara.ID != au {
		t.Fatalf("saran = %+v", saran)
	}

	reload := l.penjadwal.jumlahReload()
	resp = kirim(t, c, "POST", url+"terapkan", map[string]any{"isi": []map[string]any{{"jadwalId": saran[0].JadwalID, "audioId": au}}})
	harapKode(t, resp, 200)
	var h struct{ Diisi int }
	bacaBody(t, resp, &h)
	if h.Diisi != 1 {
		t.Fatalf("diisi = %d", h.Diisi)
	}
	if l.penjadwal.jumlahReload() == reload {
		t.Fatal("harus memuat ulang penjadwal")
	}

	resp = kirim(t, c, "GET", url+"saran", nil)
	harapKode(t, resp, 200)
	var kosong []store.SaranPengumuman
	bacaBody(t, resp, &kosong)
	if kosong == nil || len(kosong) != 0 {
		t.Fatalf("saran harus [] : %#v", kosong)
	}
}

func TestPengumuman_404Dan401(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	harapKode(t, kirim(t, c, "GET", l.srv.URL+"/api/profil/9999/pengumuman/saran", nil), 404)
	harapKode(t, kirim(t, c, "POST", l.srv.URL+"/api/profil/9999/pengumuman/terapkan", map[string]any{"isi": []any{}}), 404)

	k := l.klien()
	harapKode(t, kirim(t, k, "GET", l.srv.URL+"/api/pengumuman/cocok?keterangan=a", nil), 401)
	harapKode(t, kirim(t, k, "GET", l.srv.URL+"/api/profil/1/pengumuman/saran", nil), 401)
	harapKode(t, kirim(t, k, "POST", l.srv.URL+"/api/profil/1/pengumuman/terapkan", map[string]any{"isi": []any{}}), 401)
}

func TestPengumuman_StatusDanHapusSuara(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	profil, _ := l.st.ProfilDefault(ctx)
	js, _ := l.st.ListJenisBel(ctx)
	au := suaraUji(t, l, "Jam ke-3 dimulai")
	hari := schedule.Hari(time.Now())
	if _, err := l.st.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: hari, Jam: "08:00", JenisBelID: js[0].ID, Label: "Jam ke-3", PengumumanID: au}); err != nil {
		t.Fatal(err)
	}
	if _, err := l.st.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: hari, Jam: "09:00", JenisBelID: js[0].ID, Label: "Polos"}); err != nil {
		t.Fatal(err)
	}
	l.penjadwal.status = scheduler.Status{
		Siap: true, Tanggal: "2026-10-12", ProfilID: profil,
		Bel: []scheduler.StatusBel{
			{JadwalID: 1, Waktu: l.now.Add(time.Hour), JenisBelID: js[0].ID, Label: "Jam ke-3", PengumumanID: au},
			{JadwalID: 2, Waktu: l.now.Add(2 * time.Hour), JenisBelID: js[0].ID, Label: "Polos"},
		},
	}
	var st struct {
		Bel []struct {
			Label      string              `json:"label"`
			Pengumuman *store.SuaraRingkas `json:"pengumuman"`
		} `json:"bel"`
	}
	resp := kirim(t, c, "GET", l.srv.URL+"/api/status", nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &st)
	if len(st.Bel) != 2 {
		t.Fatalf("bel = %+v", st.Bel)
	}
	for _, b := range st.Bel {
		if b.Label == "Jam ke-3" && (b.Pengumuman == nil || b.Pengumuman.ID != au || b.Pengumuman.Nama != "Jam ke-3 dimulai") {
			t.Fatalf("pengumuman = %+v", b.Pengumuman)
		}
		if b.Label == "Polos" && b.Pengumuman != nil {
			t.Fatalf("harus null: %+v", b.Pengumuman)
		}
	}

	resp = kirim(t, c, "DELETE", l.srv.URL+"/api/audio/"+itoa(au), nil)
	harapKode(t, resp, 409)
	if p := pesanError(t, resp); !strings.Contains(p, "bel di jadwal") {
		t.Fatalf("pesan = %q", p)
	}
}
