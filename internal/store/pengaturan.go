package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

// KunciVolume menyimpan volume pemutaran 0–100.
const KunciVolume = "volume"

// Volume mengembalikan volume tersimpan, atau 100 jika belum pernah diatur.
func (s *Store) Volume(ctx context.Context) (int, error) {
	nilai, err := s.GetSetting(ctx, KunciVolume)
	if errors.Is(err, ErrTidakAda) {
		return 100, nil
	}
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(nilai)
	if err != nil {
		return 0, fmt.Errorf("volume tersimpan tidak valid %q: %w", nilai, err)
	}
	return v, nil
}

func (s *Store) SetVolume(ctx context.Context, v int) error {
	if v < 0 || v > 100 {
		return validasi("volume harus 0 sampai 100")
	}
	return s.SetSetting(ctx, KunciVolume, strconv.Itoa(v))
}
