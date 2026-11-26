import { createRouter, createWebHistory } from 'vue-router'
import { config } from '@/lib/config'

export default createRouter({
  history: createWebHistory(config.baseUrl),
  routes: [
    { path: '/', name: 'home', component: () => import('../views/Home.vue') },
    { path: '/logs', name: 'logs', component: () => import('../views/Logs.vue') },
    { path: '/status', name: 'status', component: () => import('../views/Status.vue') },
  ],
})
