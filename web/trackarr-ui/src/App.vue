<script setup>
import { onEvent } from '@/lib/socket'
import { useDisplay } from 'vuetify'
import { Toaster, toast } from 'vue-sonner'
import 'vue-sonner/style.css'

const { xs } = useDisplay()

const links = [
  { to: '/', title: 'Home', icon: 'mdi-home' },
  { to: '/status', title: 'Tracker Statuses', icon: 'mdi-access-point' },
  { to: '/logs', title: 'Logs', icon: 'mdi-folder' },
]

const levels = { info: toast.info, success: toast.success, warn: toast.warning, error: toast.error }

onEvent('alert', ({ level, title, msg }) => levels[level]?.(title || msg, { description: title ? msg : undefined }))
</script>

<template>
  <v-app>
    <v-app-bar v-if="xs">
      <v-menu>
        <template #activator="{ props }">
          <v-btn icon="mdi-menu" aria-label="Navigation menu" v-bind="props" />
        </template>
        <v-list>
          <v-list-item v-for="link in links" :key="link.to" :to="link.to" :title="link.title" />
        </v-list>
      </v-menu>
      <v-toolbar-title>Trackarr</v-toolbar-title>
    </v-app-bar>

    <v-navigation-drawer v-else rail permanent>
      <v-list density="compact" nav>
        <v-list-item class="text-center text-uppercase font-weight-black" title="T" />
        <v-divider />
        <v-list-item v-for="link in links" :key="link.to" :to="link.to" :prepend-icon="link.icon" :title="link.title" />
      </v-list>
    </v-navigation-drawer>

    <v-main>
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </v-main>

    <Toaster theme="dark" position="bottom-right" rich-colors />
  </v-app>
</template>
