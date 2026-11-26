import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'

dayjs.extend(relativeTime)

export const fromNow = (value) => dayjs(value).fromNow()
export const formatTime = (value) => dayjs(value).format('MM/DD/YY, h:mm:ss a')
export const capitalize = (value) => {
  const text = String(value ?? '')
  return text.charAt(0).toUpperCase() + text.slice(1)
}
