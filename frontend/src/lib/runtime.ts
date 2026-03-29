import { ClipboardSetText, EventsOff, EventsOn } from '../../wailsjs/runtime/runtime'
import type { DownloadTask } from '../types'

export function subscribeDownloadUpdates(handler: (task: DownloadTask) => void): () => void {
  EventsOn('download:updated', (task: DownloadTask) => {
    handler(task)
  })

  return () => {
    EventsOff('download:updated')
  }
}

export async function setClipboardText(text: string): Promise<void> {
  try {
    const copied = await ClipboardSetText(text)
    if (copied) {
      return
    }
  } catch {
    // Fall back to the browser clipboard API when runtime clipboard is unavailable.
  }

  if (!navigator.clipboard) {
    throw new Error('clipboard is unavailable')
  }

  await navigator.clipboard.writeText(text)
}
