import { createApp } from 'vue'
import { createVuetify } from 'vuetify'
import '@mdi/font/css/materialdesignicons.css'
import 'vuetify/styles'
import '@/assets/styles/trackarr.css'
import App from './App.vue'
import router from './router'
import { startSocket } from '@/lib/socket'

startSocket()

createApp(App)
  .use(router)
  .use(createVuetify({ theme: { defaultTheme: 'dark' } }))
  .mount('#app')
