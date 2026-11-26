<script setup>
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import { capitalize, fromNow } from '@/lib/format'
import { onEvent, useTopic } from '@/lib/socket'

const headers = [
  { title: 'Age', key: 'age', sortable: false },
  { title: 'Release', key: 'release', sortable: false },
  { title: 'Tracker', key: 'tracker', sortable: false },
  { title: 'PVR', key: 'pvr', sortable: false },
]

const releases = ref([])
const loading = ref(true)
const sections = [
  { title: 'Pushed Releases', approvedOnly: false },
  { title: 'Approved Releases', approvedOnly: true },
].map((section) => ({
  ...section,
  search: ref(''),
  trackers: ref([]),
  pvrs: ref([]),
}))

const toRelease = (r) => ({ age: r.CreatedAt, release: r.Name, pvr: r.PvrName, tracker: r.TrackerName, approved: r.Approved })

const rowsFor = (section) => releases.value.filter((r) => !section.approvedOnly || r.approved)

const unique = (rows, key) => [...new Set(rows.map((r) => r[key]))]

function visible(section) {
  return rowsFor(section)
    .filter((r) => section.trackers.value.length === 0 || section.trackers.value.includes(r.tracker))
    .filter((r) => section.pvrs.value.length === 0 || section.pvrs.value.includes(r.pvr))
    .sort((a, b) => new Date(b.age) - new Date(a.age))
}

useTopic('releases')
onEvent('release', (data) => releases.value.push(toRelease(data)))

onMounted(async () => {
  const { data } = await api.get('/releases')
  releases.value.push(...data.map(toRelease))
  loading.value = false
})
</script>

<template>
  <v-container fluid>
    <template v-for="section in sections" :key="section.title">
      <v-row align="center">
        <v-col cols="12" md="3">
          <h1 class="text-h5 font-weight-light">{{ section.title }}</h1>
        </v-col>
        <v-col cols="12" md="9" class="d-flex flex-wrap ga-4">
          <v-select v-model="section.trackers.value" label="Trackers" prepend-inner-icon="mdi-filter" :items="unique(rowsFor(section), 'tracker')" density="compact" multiple clearable hide-details />
          <v-select v-model="section.pvrs.value" label="PVR" prepend-inner-icon="mdi-filter" :items="unique(rowsFor(section), 'pvr')" density="compact" multiple clearable hide-details />
          <v-text-field v-model="section.search.value" label="Search" append-inner-icon="mdi-magnify" density="compact" single-line hide-details />
        </v-col>
      </v-row>
      <v-data-table class="mb-6" :search="section.search.value" :loading="loading" fixed-header :headers="headers" :items="visible(section)" :items-per-page="5">
        <template #item.age="{ item }">{{ fromNow(item.age) }}</template>
        <template #item.pvr="{ item }">{{ capitalize(item.pvr) }}</template>
      </v-data-table>
    </template>
  </v-container>
</template>
