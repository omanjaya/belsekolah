export interface Notif { id: number; jenis: 'sukses' | 'galat' | 'info'; pesan: string }

export const daftarNotif = $state<Notif[]>([])
let urut = 0

export function beriTahu(pesan: string, jenis: Notif['jenis'] = 'sukses') {
  const id = ++urut
  daftarNotif.push({ id, jenis, pesan })
  setTimeout(() => tutupNotif(id), jenis === 'galat' ? 6000 : 3000)
}

export function tutupNotif(id: number) {
  const i = daftarNotif.findIndex((n) => n.id === id)
  if (i >= 0) daftarNotif.splice(i, 1)
}

export function galat(e: unknown) {
  beriTahu(e instanceof Error ? e.message : String(e), 'galat')
}
