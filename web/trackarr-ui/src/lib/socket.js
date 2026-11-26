import { onScopeDispose } from 'vue'
import { config } from './config'

const handlers = new Map()
const topics = new Set()
let socket
let retryDelay = 1000

const dispatch = (event) => handlers.get(event.type)?.forEach((fn) => fn(event.data))

function connect() {
  socket = new WebSocket(config.websocketUrl)
  socket.onopen = () => {
    retryDelay = 1000
    topics.forEach((data) => send({ type: 'subscribe', data }))
  }
  socket.onmessage = (message) => dispatch(JSON.parse(message.data))
  socket.onclose = () => {
    setTimeout(connect, retryDelay)
    retryDelay = Math.min(retryDelay * 2, 30000)
  }
}

function send(message) {
  if (socket?.readyState === WebSocket.OPEN) socket.send(JSON.stringify(message))
}

export function startSocket() {
  if (!socket) connect()
}

// Calls fn for every event of the given type until the calling scope is disposed.
export function onEvent(type, fn) {
  if (!handlers.has(type)) handlers.set(type, new Set())
  handlers.get(type).add(fn)
  onScopeDispose(() => handlers.get(type).delete(fn))
}

// Subscribes to a server topic (re-subscribing after reconnects) until the calling scope is disposed.
export function useTopic(topic) {
  topics.add(topic)
  send({ type: 'subscribe', data: topic })
  onScopeDispose(() => {
    topics.delete(topic)
    send({ type: 'unsubscribe', data: topic })
  })
}
