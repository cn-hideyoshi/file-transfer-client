export type DownloadState = 'queued' | 'running' | 'done' | 'failed' | 'canceled'

export interface Settings {
  last_server_url: string
  last_remote_path: string
  default_download_dir: string
  recent_servers: string[]
}

export interface AppState {
  settings: Settings
  downloads: DownloadTask[]
}

export interface ConnectionState {
  server_url: string
  checked_at: string
  settings: Settings
}

export interface Entry {
  name: string
  path: string
  is_dir: boolean
  size: number
  last_modified: string
}

export interface Directory {
  path: string
  entries: Entry[]
}

export interface DownloadRequest {
  remote_path: string
  save_path: string
  overwrite: boolean
}

export interface DownloadTask {
  id: string
  remote_path: string
  save_path: string
  temp_path?: string
  total_bytes: number
  written_bytes: number
  state: DownloadState
  error_message?: string
  started_at?: string
  finished_at?: string
}
