/** Ambil bagian path saja (tanpa query/hash) dari alamat relatif atau penuh. */
export function pathDari(ke: string): string {
  return new URL(ke, location.origin).pathname
}

export const rute = $state({ path: location.pathname })

window.addEventListener('popstate', () => {
  rute.path = location.pathname
})

export function pindah(ke: string) {
  if (ke === location.pathname + location.search) return
  history.pushState(null, '', ke)
  rute.path = pathDari(ke)
  window.scrollTo(0, 0)
}

/** Pasang sebagai onclick pada <a href="/..."> agar navigasi tanpa memuat ulang halaman. */
export function tautan(e: MouseEvent) {
  if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return
  const href = (e.currentTarget as HTMLAnchorElement).getAttribute('href')
  if (!href || !href.startsWith('/')) return
  e.preventDefault()
  pindah(href)
}
