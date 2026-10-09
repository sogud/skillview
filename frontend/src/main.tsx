import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import './styles.css'
import { LocaleProvider } from './locales'

document.documentElement.dataset.inputMode = 'pointer'
window.addEventListener('pointerdown', () => { document.documentElement.dataset.inputMode = 'pointer' }, true)
window.addEventListener('keydown', (event) => { if (event.key === 'Tab' || event.key.startsWith('Arrow')) document.documentElement.dataset.inputMode = 'keyboard' }, true)

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <LocaleProvider><App /></LocaleProvider>
  </StrictMode>,
)
