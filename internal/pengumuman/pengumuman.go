// Package pengumuman mencocokkan keterangan bel dengan nama suara di pustaka
// untuk mengisi pengumuman secara otomatis. Murni, tanpa database.
package pengumuman

import "strings"

type Kandidat struct {
	ID   int64
	Nama string
}

var pemisah = strings.NewReplacer("-", " ", "_", " ", ".", " ", ",", " ", ":", " ", "(", " ", ")", " ", "/", " ")

// Normal: huruf kecil, tanda baca umum menjadi spasi, spasi dirapatkan.
func Normal(s string) string {
	return strings.Join(strings.Fields(pemisah.Replace(strings.ToLower(s))), " ")
}

// padanan menerjemahkan keterangan umum ke awal nama suara yang lazim.
var padanan = map[string]string{
	"pulang":        "akhir pelajaran",
	"upacara":       "upacara bendera",
	"masuk":         "bel masuk sekolah",
	"ujian selesai": "ujian waktu telah habis",
}

func tingkat(n, k string) int {
	switch {
	case n == k:
		return 1
	case n == k+" dimulai":
		return 2
	case strings.HasPrefix(n, k+" "):
		return 3
	}
	return 0
}

// Cocokkan mengembalikan suara yang paling cocok dengan keterangan.
// Tidak menebak: bila tingkat terbaik punya lebih dari satu kandidat, hasilnya (0, false).
func Cocokkan(keterangan string, suara []Kandidat) (int64, bool) {
	k := Normal(keterangan)
	if k == "" {
		return 0, false
	}
	if p, ok := padanan[k]; ok {
		k = p
	}
	terbaik, jumlah := 0, 0
	var id int64
	for _, s := range suara {
		t := tingkat(Normal(s.Nama), k)
		switch {
		case t == 0:
		case terbaik == 0 || t < terbaik:
			terbaik, jumlah, id = t, 1, s.ID
		case t == terbaik:
			jumlah++
		}
	}
	if jumlah != 1 {
		return 0, false
	}
	return id, true
}
