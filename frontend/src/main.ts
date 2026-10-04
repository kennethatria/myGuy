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
const umamiWebsiteId = import.meta.env.VITE_UMAMI_WEBSITE_ID
if (umamiWebsiteId) {
  const script = document.createElement('script')
  script.defer = true
  script.src = '/umami/script.js'
  script.dataset.websiteId = umamiWebsiteId
  document.head.appendChild(script)
}
