import { EventsOff, EventsOn } from '../../wailsjs/runtime/runtime'
import type { DownloadTask } from '../types'

export function subscribeDownloadUpdates(handler: (task: DownloadTask) => void): () => void {
  EventsOn('download:updated', (task: DownloadTask) => {
    handler(task)
  })

  return () => {
    EventsOff('download:updated')
  }
}
