import { beforeEach, describe, expect, it, vi } from 'vitest'

const ORIGIN = 'http://localhost'
const lok = { pathname: '/', search: '', origin: ORIGIN }
const dengar: Record<string, () => void> = {}

function pasangPeramban() {
  lok.pathname = '/'
  lok.search = ''
  vi.stubGlobal('location', lok)
  vi.stubGlobal('window', {
    addEventListener: (n: string, f: () => void) => { dengar[n] = f },
    scrollTo: () => {},
  })
  vi.stubGlobal('history', {
    pushState: (_d: unknown, _t: string, url: string) => {
      const u = new URL(url, ORIGIN)
      lok.pathname = u.pathname
      lok.search = u.search
    },
  })
}

describe('router', () => {
  beforeEach(() => {
    vi.resetModules()
    pasangPeramban()
  })

  it('pindah dengan query hanya menyimpan pathname di rute.path', async () => {
    const { pindah, rute } = await import('./router.svelte')
    pindah('/jadwal?profil=5')
    expect(rute.path).toBe('/jadwal')
    expect(lok.search).toBe('?profil=5')
  })

  it('pindah dengan hash menyimpan pathname saja', async () => {
    const { pindah, rute } = await import('./router.svelte')
    pindah('/libur#baru')
    expect(rute.path).toBe('/libur')
  })

  it('popstate memakai pathname', async () => {
    const { rute } = await import('./router.svelte')
    lok.pathname = '/profil'
    lok.search = '?x=1'
    dengar.popstate()
    expect(rute.path).toBe('/profil')
  })

  it('pathDari membuang query', async () => {
    const { pathDari } = await import('./router.svelte')
    expect(pathDari('/jadwal?profil=5')).toBe('/jadwal')
  })
})
