interface Permintaan {
  buka: boolean
  judul: string
  pesan: string
  label: string
  jawab: ((ya: boolean) => void) | null
}

export const permintaanKonfirmasi = $state<Permintaan>({ buka: false, judul: '', pesan: '', label: 'Hapus', jawab: null })

/** Menampilkan dialog konfirmasi; resolve true jika pengguna menekan tombol aksi. */
export function konfirmasi(judul: string, pesan: string, label = 'Hapus'): Promise<boolean> {
  permintaanKonfirmasi.jawab?.(false)
  return new Promise((jawab) => {
    Object.assign(permintaanKonfirmasi, { buka: true, judul, pesan, label, jawab })
  })
}
