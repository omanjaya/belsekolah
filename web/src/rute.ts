import type { Component } from 'svelte'
import { BellRing, CalendarClock, CalendarOff, History, House, Layers, LayoutTemplate, Music, Settings } from '@lucide/svelte'
import Beranda from './pages/Beranda.svelte'
import Jadwal from './pages/Jadwal.svelte'
import JenisBel from './pages/JenisBel.svelte'
import Libur from './pages/Libur.svelte'
import Pengaturan from './pages/Pengaturan.svelte'
import ProfilPeriode from './pages/ProfilPeriode.svelte'
import PustakaSuara from './pages/PustakaSuara.svelte'
import Riwayat from './pages/Riwayat.svelte'
import Template from './pages/Template.svelte'
import TidakDitemukan from './pages/TidakDitemukan.svelte'

export interface ItemMenu {
  path: string
  label: string
  labelPanjang?: string // nama di sidebar/laci bila berbeda dari label bar bawah
  ikon: Component
  utama: boolean // tampil di bar bawah HP
  halaman: Component
}

export const menu: ItemMenu[] = [
  { path: '/', label: 'Beranda', ikon: House, utama: true, halaman: Beranda },
  { path: '/jadwal', label: 'Jadwal', ikon: CalendarClock, utama: true, halaman: Jadwal },
  { path: '/profil', label: 'Profil & Periode', ikon: Layers, utama: false, halaman: ProfilPeriode },
  { path: '/template', label: 'Template', ikon: LayoutTemplate, utama: false, halaman: Template },
  { path: '/libur', label: 'Libur', ikon: CalendarOff, utama: false, halaman: Libur },
  { path: '/jenis-bel', label: 'Jenis Bel', ikon: BellRing, utama: false, halaman: JenisBel },
  { path: '/suara', label: 'Suara', labelPanjang: 'Pustaka Suara', ikon: Music, utama: true, halaman: PustakaSuara },
  { path: '/riwayat', label: 'Riwayat', ikon: History, utama: true, halaman: Riwayat },
  { path: '/pengaturan', label: 'Pengaturan', ikon: Settings, utama: false, halaman: Pengaturan },
]

export function halamanUntuk(path: string): Component {
  return menu.find((m) => m.path === path)?.halaman ?? TidakDitemukan
}
