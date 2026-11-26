<script setup>
import { computed, nextTick, ref } from 'vue'
import { formatTime } from '@/lib/format'
import { onEvent, useTopic } from '@/lib/socket'

const headers = [
  { title: 'Timestamp', key: 'time', width: '10%' },
  { title: 'Level', key: 'level', width: '7%' },
  { title: 'Component', key: 'component', width: '7%' },
  { title: 'Message', key: 'message', width: '76%' },
]
const logLevels = [
  { level: 'TRACE', color: '#808080' },
  { level: 'DEBUG', color: '#00AAAA' },
  { level: 'INFO', color: '#00AA00' },
  { level: 'WARN', color: '#AAAA00' },
  { level: 'ERROR', color: '#AA0000' },
]
const maxStored = 500

function loadStored() {
  try {
    return JSON.parse(localStorage.getItem('logs')) ?? []
  } catch {
    return []
  }
}

const table = ref(null)
const messages = ref(loadStored())
const search = ref('')
const itemsPerPage = ref(15)
const sortBy = ref([])
const levelFilter = ref([])
const componentFilter = ref([])

const components = computed(() => [...new Set(messages.value.map((m) => m.component))])
const filtered = computed(() =>
  messages.value
    .filter((m) => levelFilter.value.length === 0 || levelFilter.value.includes(m.level.toUpperCase()))
    .filter((m) => componentFilter.value.length === 0 || componentFilter.value.includes(m.component)),
)

const levelColor = (level) => logLevels.find((l) => l.level === level.toUpperCase())?.color ?? ''

function clearLogs() {
  messages.value = []
  localStorage.removeItem('logs')
}

function scroller() {
  return table.value?.$el.querySelector('.v-table__wrapper')
}

function nearBottom(el) {
  const scrollable = el.scrollHeight - el.clientHeight
  return scrollable <= 0 || (el.scrollTop / scrollable) * 100 >= 90
}

useTopic('logs')
onEvent('log', async (data) => {
  const el = scroller()
  const follow = itemsPerPage.value === -1 && sortBy.value.length === 0 && el && nearBottom(el)

  messages.value.push(data)
  localStorage.setItem('logs', JSON.stringify(messages.value.slice(-maxStored)))

  if (follow) {
    await nextTick()
    el.scrollTop = el.scrollHeight
  }
})
</script>

<template>
  <v-container fluid>
    <v-row align="center">
      <v-col cols="12" md="3">
        <h1 class="text-h5 font-weight-light">System Logs</h1>
      </v-col>
      <v-col cols="12" md="9" class="d-flex flex-wrap ga-4">
        <v-select v-model="levelFilter" label="Log Level" prepend-inner-icon="mdi-filter" :items="logLevels.map((l) => l.level)" density="compact" multiple clearable hide-details />
        <v-select v-model="componentFilter" label="Component" prepend-inner-icon="mdi-filter" :items="components" density="compact" multiple clearable hide-details />
        <v-text-field v-model="search" label="Search" append-inner-icon="mdi-magnify" density="compact" single-line hide-details />
        <v-btn @click="clearLogs">Clear Logs</v-btn>
      </v-col>
    </v-row>
    <v-data-table
      ref="table"
      v-model:items-per-page="itemsPerPage"
      v-model:sort-by="sortBy"
      :search="search"
      :headers="headers"
      :items="filtered"
      :items-per-page-options="[10, 15, 50, 100, { value: -1, title: 'All' }]"
      height="75vh"
      fixed-header
    >
      <template #item.time="{ item }">{{ formatTime(item.time) }}</template>
      <template #item.level="{ item }">
        <span :style="{ color: levelColor(item.level) }">{{ item.level.toUpperCase() }}</span>
      </template>
    </v-data-table>
  </v-container>
</template>
