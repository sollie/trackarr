<script setup>
import { computed, onMounted, onScopeDispose, ref } from 'vue'
import { toast } from 'vue-sonner'
import StatusIndicator from '@/components/StatusIndicator.vue'
import { api } from '@/lib/api'
import { config } from '@/lib/config'
import { capitalize, fromNow } from '@/lib/format'

const headers = [
  { title: 'Tracker', key: 'tracker', sortable: false },
  { title: 'Status', key: 'status', sortable: false },
  { title: 'Last Announced', key: 'last_announced', sortable: false },
]
const statusOptions = [
  { title: 'Online', value: 'online' },
  { title: 'Offline', value: 'offline' },
]

const trackers = ref([])
const search = ref('')
const statusFilter = ref(null)

const filtered = computed(() => trackers.value.filter((item) => !statusFilter.value || item.status === statusFilter.value))

async function fetchStatuses() {
  const { data } = await api.get('/irc/status')
  trackers.value = Object.entries(data).map(([tracker, value]) => ({
    tracker,
    status: value.connected === true ? 'online' : 'offline',
    last_joined: value.last_joined || null,
    last_announced: value.last_announced || null,
  }))
}

async function checkForUpdate() {
  const { data } = await api.get('/update/status')
  toast(data.update_available ? `Newer version available: ${data.latest_version}` : "You're already on the latest version!", {
    position: 'top-right',
  })
}

onMounted(() => {
  fetchStatuses()
  const interval = setInterval(fetchStatuses, 60000)
  onScopeDispose(() => clearInterval(interval))
})
</script>

<template>
  <v-container fluid>
    <v-row align="center">
      <v-col cols="12" sm="4">
        <h1 class="text-h5 font-weight-light">Tracker Status</h1>
      </v-col>
      <v-col cols="12" sm="8" class="d-flex ga-4">
        <v-select v-model="statusFilter" label="Tracker Status" prepend-inner-icon="mdi-filter" :items="statusOptions" density="compact" clearable hide-details />
        <v-text-field v-model="search" label="Search" append-inner-icon="mdi-magnify" density="compact" single-line hide-details />
      </v-col>
    </v-row>
    <v-data-table :search="search" :headers="headers" :items="filtered" :items-per-page="15">
      <template #item.status="{ item }">
        <status-indicator v-if="item.status" :online="item.status === 'online'" />
        {{ capitalize(item.status) }}
        <i v-if="item.status === 'online' && item.last_joined">(joined {{ fromNow(item.last_joined) }})</i>
      </template>
      <template #item.last_announced="{ item }">
        {{ item.last_announced ? capitalize(fromNow(item.last_announced)) : 'Never' }}
      </template>
    </v-data-table>
    <v-footer class="justify-end ga-4">
      <v-btn color="secondary" @click="checkForUpdate">Check For Update</v-btn>
      <strong>Trackarr {{ config.version }}</strong>
    </v-footer>
  </v-container>
</template>
