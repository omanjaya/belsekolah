import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'
import { beriTahu } from './lib/notif.svelte'
import { konfirmasi } from './lib/konfirmasi.svelte'

// Hook khusus tes E2E; hanya ada jika dibangun dengan VITE_E2E=1 (hilang dari build produksi).
if (import.meta.env.VITE_E2E) {
  ;(window as unknown as Record<string, unknown>).__bel = { beriTahu, konfirmasi }
}

mount(App, { target: document.getElementById('app')! })
