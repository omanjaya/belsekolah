package store

import "context"

// BackupKe menyalin database secara konsisten ke path. File tujuan tidak boleh sudah ada.
func (s *Store) BackupKe(ctx context.Context, path string) error {
	_, err := s.db.ExecContext(ctx, `VACUUM INTO ?`, path)
	return err
}
