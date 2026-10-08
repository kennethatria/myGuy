import '@fontsource/dm-sans/400.css'
import '@fontsource/dm-sans/500.css'
import '@fontsource/dm-sans/600.css'
import '@fontsource/dm-sans/700.css'
// Handwriting, used sparingly on Home's notes
import '@fontsource/caveat/600.css'
import './assets/base.css'
import './assets/custom.css'
import '@fortawesome/fontawesome-free/css/all.min.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'

const app = createApp(App)

app.use(createPinia())
app.use(router)

app.mount('#app')

// Self-hosted Umami analytics, proxied by nginx under /umami/.  Only loaded
// when a website ID is provided at build time (VITE_UMAMI_WEBSITE_ID).
// recorder.js adds heatmaps (click positions and scroll depth, no page
// content) when they are switched on for the website in Umami; it waits for
// script.js's session.  Keep replays off: they would record chats.
const umamiWebsiteId = import.meta.env.VITE_UMAMI_WEBSITE_ID
if (umamiWebsiteId) {
  for (const src of ['/umami/script.js', '/umami/recorder.js']) {
    const script = document.createElement('script')
    script.defer = true
    script.src = src
    script.dataset.websiteId = umamiWebsiteId
    document.head.appendChild(script)
  }
}
