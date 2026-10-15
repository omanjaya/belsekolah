package formatjadwal

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestBaca_TanpaKunciJadwalJadiSliceKosong(t *testing.T) {
	d, err := Baca([]byte(`{"format":"` + Format + `","versi":` + strconv.Itoa(Versi) + `,"nama":"X"}`))
	if err != nil {
		t.Fatal(err)
	}
	if d.Jadwal == nil || d.JenisBel == nil {
		t.Fatalf("jadwal/jenisBel harus slice kosong, bukan nil: %+v", d)
	}
	b, _ := json.Marshal(d)
	if strings.Contains(string(b), "null") {
		t.Fatalf("hasil JSON memuat null: %s", b)
	}
}
