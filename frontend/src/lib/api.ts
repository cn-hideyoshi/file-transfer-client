import {
  Bootstrap,
  Browse,
  CancelDownload,
  Connect,
  OpenLocalPath,
  PickDefaultDownloadDir,
  PickDownloadLocation,
  SetDefaultDownloadDir,
  StartDownload,
} from '../../wailsjs/go/main/App'
import type {
  AppState,
  ConnectionState,
  Directory,
  DownloadRequest,
  DownloadTask,
  Settings,
} from '../types'

export const api = {
  bootstrap: () => Bootstrap() as Promise<AppState>,
  connect: (serverURL: string) => Connect(serverURL) as Promise<ConnectionState>,
  browse: (remotePath: string) => Browse(remotePath) as Promise<Directory>,
  pickDownloadLocation: (remotePath: string) => PickDownloadLocation(remotePath) as Promise<string>,
  startDownload: (request: DownloadRequest) => StartDownload(request) as Promise<DownloadTask>,
  cancelDownload: (taskID: string) => CancelDownload(taskID) as Promise<void>,
  openLocalPath: (targetPath: string) => OpenLocalPath(targetPath) as Promise<void>,
  pickDefaultDownloadDir: () => PickDefaultDownloadDir() as Promise<string>,
  setDefaultDownloadDir: (dir: string) => SetDefaultDownloadDir(dir) as Promise<Settings>,
}
