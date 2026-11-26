const env = import.meta.env

const baseUrl = env.VITE_BASE_URL ?? window.baseurl ?? '/'
const apiKey = env.VITE_API_KEY ?? window.apikey
const version = env.VITE_API_KEY ? 'Local Build' : window.version

const join = (...parts) => '/' + parts.map((p) => p.replace(/^\/+|\/+$/g, '')).filter(Boolean).join('/')

const apiUrl = env.VITE_API_URL ?? new URL(join(baseUrl, 'api'), window.location.href).href

let websocketUrl = env.VITE_WEBSOCKET
if (!websocketUrl) {
  const url = new URL(join(baseUrl, 'api/ws'), window.location.href)
  url.protocol = url.protocol.replace('http', 'ws')
  url.searchParams.set('apikey', apiKey)
  websocketUrl = url.href
}

export const config = { baseUrl, apiKey, version, apiUrl, websocketUrl }
