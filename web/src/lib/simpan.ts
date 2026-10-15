/** localStorage aman: gagal (mode privat, diblokir) tidak pernah melempar galat. */
export function baca<T>(kunci: string, bawaan: T): T {
  try {
    const mentah = localStorage.getItem(kunci)
    return mentah === null ? bawaan : (JSON.parse(mentah) as T)
  } catch {
    return bawaan
  }
}

export function tulis(kunci: string, nilai: unknown): void {
  try {
    localStorage.setItem(kunci, JSON.stringify(nilai))
  } catch {
    /* penyimpanan tidak tersedia */
  }
}
