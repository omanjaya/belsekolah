package store

import (
	"context"
	"errors"
)

// KunciZonaWaktu menyimpan nama IANA zona waktu sekolah (diutamakan di atas flag -tz).
const KunciZonaWaktu = "zona_waktu"

// ZonaWaktu mengembalikan "" bila belum pernah diatur.
func (s *Store) ZonaWaktu(ctx context.Context) (string, error) {
	z, err := s.GetSetting(ctx, KunciZonaWaktu)
	if errors.Is(err, ErrTidakAda) {
		return "", nil
	}
	return z, err
}

func (s *Store) SetZonaWaktu(ctx context.Context, iana string) error {
	return s.SetSetting(ctx, KunciZonaWaktu, iana)
}
