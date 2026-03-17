<template>
  <div class="shell">
    <div class="shell__glow shell__glow--left"></div>
    <div class="shell__glow shell__glow--right"></div>

    <div class="app">
      <ServerSidebar
        :server-input="serverInput"
        :recent-servers="settings.recent_servers"
        :connected-server="connectedServer"
        :default-download-dir="settings.default_download_dir"
        :is-connecting="connecting"
        :checked-at="formatStamp(connectionCheckedAt)"
        @update:server-input="serverInput = $event"
        @connect="handleConnect()"
        @use-recent="handleRecent"
        @open-settings="settingsOpen = true"
      />

      <main class="workspace">
        <header class="workspace__hero">
          <div>
            <span class="workspace__eyebrow">Remote file browser</span>
            <h2>Browse once. Pull files fast.</h2>
            <p>
              连接一个 `file-transfer` 服务端，在桌面界面里浏览目录、挑选文件、跟踪下载任务。
            </p>
          </div>
          <div class="workspace__pill" :class="{ 'workspace__pill--live': isConnected }">
            <strong>{{ isConnected ? 'Live link' : 'Waiting' }}</strong>
            <span>{{ isConnected ? connectedServer : 'Connect to get started' }}</span>
          </div>
        </header>

        <div v-if="banner" class="banner" :class="`banner--${banner.tone}`">
          {{ banner.message }}
        </div>

        <section v-if="!isConnected" class="empty-stage">
          <article class="empty-stage__panel">
            <span class="empty-stage__eyebrow">Start here</span>
            <h3>先连接服务端，再开始浏览和下载。</h3>
            <p>
              左侧输入 `http://127.0.0.1:8080` 这样的地址，连通后右侧就会切换成远程文件浏览器。
            </p>
          </article>
          <article class="empty-stage__panel empty-stage__panel--steps">
            <div>
              <span>01</span>
              <p>填入服务端地址并发起健康检查。</p>
            </div>
            <div>
              <span>02</span>
              <p>浏览 `/files` 下的目录，选中文件查看详情。</p>
            </div>
            <div>
              <span>03</span>
              <p>选择本地保存位置，下载任务会在右侧面板持续更新。</p>
            </div>
          </article>
        </section>

        <section v-else class="browser-layout">
          <div class="browser-layout__main card">
            <FileTable
              :entries="directory.entries"
              :current-path="directory.path"
              :loading="browsing || loading"
              :selected-path="selectedPath"
              @select="selectedPath = $event.path"
              @open="handleOpenEntry"
              @navigate="loadDirectory"
              @refresh="loadDirectory(directory.path || '/')"
            />
          </div>

          <aside class="browser-layout__side">
            <section class="card detail-card">
              <span class="detail-card__eyebrow">Selection</span>
              <template v-if="selectedEntry">
                <h3>{{ selectedEntry.name }}</h3>
                <dl class="detail-card__meta">
                  <div>
                    <dt>Remote path</dt>
                    <dd>{{ selectedEntry.path }}</dd>
                  </div>
                  <div>
                    <dt>Kind</dt>
                    <dd>{{ selectedEntry.is_dir ? 'Folder' : 'File' }}</dd>
                  </div>
                  <div>
                    <dt>Size</dt>
                    <dd>{{ formatSize(selectedEntry.size, selectedEntry.is_dir) }}</dd>
                  </div>
                  <div>
                    <dt>Modified</dt>
                    <dd>{{ formatStamp(selectedEntry.last_modified) }}</dd>
                  </div>
                </dl>
                <div class="detail-card__actions">
                  <button type="button" @click="handleOpenEntry(selectedEntry)">
                    {{ selectedEntry.is_dir ? 'Enter folder' : 'Download file' }}
                  </button>
                  <button class="detail-card__ghost" type="button" @click="copyRemotePath()">
                    Copy path
                  </button>
                </div>
              </template>
              <template v-else>
                <h3>No item selected</h3>
                <p class="detail-card__hint">点击目录或文件行后，这里会显示更细的路径和下载动作。</p>
              </template>
            </section>

            <section class="card">
              <DownloadPanel
                :tasks="tasks"
                @cancel="handleCancelDownload"
                @open-folder="handleOpenFolder"
              />
            </section>
          </aside>
        </section>
      </main>
    </div>

    <div v-if="settingsOpen" class="modal" @click.self="settingsOpen = false">
      <div class="modal__card">
        <header class="modal__header">
          <div>
            <span class="workspace__eyebrow">Preferences</span>
            <h3>Default download folder</h3>
          </div>
          <button class="modal__close" type="button" @click="settingsOpen = false">Close</button>
        </header>

        <label class="modal__label" for="download-folder">Folder</label>
        <input
          id="download-folder"
          v-model="draftDownloadDir"
          class="modal__input"
          placeholder="Choose a local folder"
          spellcheck="false"
        />

        <div class="modal__actions">
          <button type="button" @click="pickDefaultFolder">Browse</button>
          <button class="modal__primary" type="button" :disabled="savingSettings" @click="saveSettings">
            {{ savingSettings ? 'Saving...' : 'Save' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import DownloadPanel from './components/DownloadPanel.vue'
import FileTable from './components/FileTable.vue'
import ServerSidebar from './components/ServerSidebar.vue'
import { api } from './lib/api'
import { subscribeDownloadUpdates } from './lib/runtime'
import type { AppState, Directory, DownloadTask, Entry, Settings } from './types'

type BannerTone = 'info' | 'success' | 'danger'

const emptySettings: Settings = {
  last_server_url: '',
  last_remote_path: '/',
  default_download_dir: '',
  recent_servers: [],
}

const loading = ref(true)
const connecting = ref(false)
const browsing = ref(false)
const savingSettings = ref(false)

const serverInput = ref('')
const connectedServer = ref('')
const connectionCheckedAt = ref('')
const settings = ref<Settings>({ ...emptySettings })
const draftDownloadDir = ref('')
const settingsOpen = ref(false)

const directory = ref<Directory>({ path: '/', entries: [] })
const selectedPath = ref('')
const tasks = ref<DownloadTask[]>([])

const banner = ref<{ tone: BannerTone; message: string } | null>(null)

let unsubscribe: (() => void) | undefined

const isConnected = computed(() => Boolean(connectedServer.value))
const selectedEntry = computed<Entry | null>(() => {
  return directory.value.entries.find((entry) => entry.path === selectedPath.value) || null
})

onMounted(async () => {
  try {
    const state = (await api.bootstrap()) as AppState
    settings.value = state.settings
    draftDownloadDir.value = state.settings.default_download_dir
    serverInput.value = state.settings.last_server_url
    tasks.value = state.downloads
  } catch (error) {
    showBanner(errorMessage(error), 'danger')
  } finally {
    loading.value = false
  }

  unsubscribe = subscribeDownloadUpdates((task) => {
    upsertTask(task)
  })
})

onBeforeUnmount(() => {
  unsubscribe?.()
})

async function handleConnect(nextServer?: string): Promise<void> {
  const target = (nextServer ?? serverInput.value).trim()
  if (!target) {
    showBanner('Please enter a server URL first.', 'danger')
    return
  }

  connecting.value = true
  try {
    const state = await api.connect(target)
    connectedServer.value = state.server_url
    connectionCheckedAt.value = state.checked_at
    settings.value = state.settings
    draftDownloadDir.value = state.settings.default_download_dir
    serverInput.value = state.server_url
    selectedPath.value = ''
    showBanner('Connection established.', 'success')
    await loadDirectory('/')
  } catch (error) {
    showBanner(errorMessage(error), 'danger')
  } finally {
    connecting.value = false
  }
}

function handleRecent(serverURL: string): void {
  serverInput.value = serverURL
  void handleConnect(serverURL)
}

async function loadDirectory(remotePath = '/'): Promise<void> {
  browsing.value = true
  try {
    const next = await api.browse(remotePath)
    directory.value = next
    if (!next.entries.some((entry) => entry.path === selectedPath.value)) {
      selectedPath.value = next.entries[0]?.path || ''
    }
  } catch (error) {
    showBanner(errorMessage(error), 'danger')
  } finally {
    browsing.value = false
  }
}

async function handleOpenEntry(entry: Entry): Promise<void> {
  if (entry.is_dir) {
    await loadDirectory(entry.path)
    return
  }

  const savePath = await api.pickDownloadLocation(entry.path)
  if (!savePath) {
    return
  }

  await queueDownload(entry, savePath, false)
}

async function queueDownload(entry: Entry, savePath: string, overwrite: boolean): Promise<void> {
  try {
    const task = await api.startDownload({
      remote_path: entry.path,
      save_path: savePath,
      overwrite,
    })
    upsertTask(task)
    showBanner(`Queued ${entry.name}.`, 'info')
  } catch (error) {
    const message = errorMessage(error)
    if (!overwrite && message.includes('already exists')) {
      const confirmed = window.confirm(`${entry.name} already exists at that location. Replace it?`)
      if (confirmed) {
        await queueDownload(entry, savePath, true)
      }
      return
    }
    showBanner(message, 'danger')
  }
}

async function handleCancelDownload(task: DownloadTask): Promise<void> {
  try {
    await api.cancelDownload(task.id)
    showBanner(`Canceled ${tailName(task.remote_path)}.`, 'info')
  } catch (error) {
    showBanner(errorMessage(error), 'danger')
  }
}

async function handleOpenFolder(task: DownloadTask): Promise<void> {
  try {
    await api.openLocalPath(parentDirectory(task.save_path))
  } catch (error) {
    showBanner(errorMessage(error), 'danger')
  }
}

async function copyRemotePath(): Promise<void> {
  if (!selectedEntry.value) {
    return
  }

  try {
    await navigator.clipboard.writeText(selectedEntry.value.path)
    showBanner('Remote path copied.', 'success')
  } catch {
    showBanner('Copy path failed in the current environment.', 'danger')
  }
}

async function pickDefaultFolder(): Promise<void> {
  try {
    const next = await api.pickDefaultDownloadDir()
    if (next) {
      draftDownloadDir.value = next
    }
  } catch (error) {
    showBanner(errorMessage(error), 'danger')
  }
}

async function saveSettings(): Promise<void> {
  if (!draftDownloadDir.value.trim()) {
    showBanner('Please choose a download folder.', 'danger')
    return
  }

  savingSettings.value = true
  try {
    const next = await api.setDefaultDownloadDir(draftDownloadDir.value.trim())
    settings.value = next
    draftDownloadDir.value = next.default_download_dir
    settingsOpen.value = false
    showBanner('Default folder updated.', 'success')
  } catch (error) {
    showBanner(errorMessage(error), 'danger')
  } finally {
    savingSettings.value = false
  }
}

function upsertTask(next: DownloadTask): void {
  const index = tasks.value.findIndex((task) => task.id === next.id)
  if (index === -1) {
    tasks.value = [next, ...tasks.value]
    return
  }

  const copy = [...tasks.value]
  copy[index] = next
  tasks.value = copy
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message
  }
  return String(error)
}

function showBanner(message: string, tone: BannerTone): void {
  banner.value = { message, tone }
}

function formatStamp(value: string): string {
  if (!value) {
    return ''
  }
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

function formatSize(size: number, isDir: boolean): string {
  if (isDir) {
    return '-'
  }
  if (size < 1024) {
    return `${size} B`
  }
  const units = ['KB', 'MB', 'GB', 'TB']
  let value = size
  for (const unit of units) {
    value /= 1024
    if (value < 1024) {
      return `${value.toFixed(1)} ${unit}`
    }
  }
  return `${value.toFixed(1)} PB`
}

function tailName(input: string): string {
  const clean = input.replace(/\/+$/, '')
  const parts = clean.split('/')
  return parts.at(-1) || input
}

function parentDirectory(filePath: string): string {
  const slashIndex = Math.max(filePath.lastIndexOf('/'), filePath.lastIndexOf('\\'))
  if (slashIndex <= 0) {
    return filePath
  }
  return filePath.slice(0, slashIndex)
}
</script>

<style scoped>
.shell {
  position: relative;
  min-height: 100vh;
  overflow: hidden;
}

.shell__glow {
  position: absolute;
  width: 420px;
  height: 420px;
  border-radius: 50%;
  filter: blur(18px);
  opacity: 0.55;
}

.shell__glow--left {
  top: -140px;
  left: -120px;
  background: rgba(94, 199, 189, 0.18);
}

.shell__glow--right {
  right: -150px;
  bottom: -160px;
  background: rgba(37, 127, 181, 0.12);
}

.app {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  min-height: 100vh;
}

.workspace {
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr);
  gap: 20px;
  padding: 28px;
}

.workspace__hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 24px 26px;
  border-radius: 30px;
  background: rgba(255, 255, 255, 0.62);
  border: 1px solid rgba(255, 255, 255, 0.55);
  box-shadow: 0 24px 50px rgba(18, 50, 71, 0.08);
}

.workspace__hero h2 {
  margin: 8px 0 10px;
  font-size: 2rem;
}

.workspace__hero p {
  margin: 0;
  max-width: 760px;
  color: #5b7482;
  line-height: 1.7;
}

.workspace__eyebrow {
  display: inline-flex;
  color: #1b7f83;
  font-size: 0.78rem;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.workspace__pill {
  display: grid;
  gap: 8px;
  min-width: 260px;
  padding: 18px;
  border-radius: 22px;
  background: rgba(18, 50, 71, 0.06);
  color: #47606d;
}

.workspace__pill strong {
  font-size: 1rem;
}

.workspace__pill span {
  word-break: break-word;
}

.workspace__pill--live {
  background: rgba(27, 127, 131, 0.12);
  color: #155f67;
}

.banner {
  padding: 14px 18px;
  border-radius: 18px;
  font-weight: 600;
}

.banner--info {
  background: #e8f3f8;
  color: #24556f;
}

.banner--success {
  background: #e3f3e8;
  color: #2f7e53;
}

.banner--danger {
  background: #fbe8e8;
  color: #a24b4b;
}

.empty-stage {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(0, 0.8fr);
  gap: 20px;
}

.empty-stage__panel {
  padding: 28px;
  border-radius: 30px;
  background: rgba(255, 255, 255, 0.66);
  border: 1px solid rgba(255, 255, 255, 0.62);
  box-shadow: 0 24px 50px rgba(18, 50, 71, 0.08);
}

.empty-stage__panel h3 {
  margin: 10px 0 12px;
  font-size: 1.6rem;
}

.empty-stage__panel p {
  margin: 0;
  color: #5b7482;
  line-height: 1.8;
}

.empty-stage__eyebrow {
  display: inline-flex;
  padding: 8px 12px;
  border-radius: 999px;
  background: rgba(27, 127, 131, 0.12);
  color: #1b7f83;
}

.empty-stage__panel--steps {
  display: grid;
  gap: 16px;
}

.empty-stage__panel--steps div {
  display: grid;
  gap: 8px;
  padding: 16px;
  border-radius: 22px;
  background: rgba(20, 78, 110, 0.05);
}

.empty-stage__panel--steps span {
  color: #1b7f83;
  font-weight: 700;
}

.browser-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) 360px;
  gap: 20px;
  min-height: 0;
}

.browser-layout__main,
.browser-layout__side {
  min-height: 0;
}

.browser-layout__side {
  display: grid;
  gap: 20px;
}

.card {
  padding: 22px;
  border-radius: 30px;
  background: rgba(255, 255, 255, 0.66);
  border: 1px solid rgba(255, 255, 255, 0.64);
  box-shadow: 0 24px 50px rgba(18, 50, 71, 0.08);
}

.detail-card {
  display: grid;
  gap: 14px;
}

.detail-card__eyebrow {
  display: inline-flex;
  color: #6d7f89;
  font-size: 0.78rem;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.detail-card h3 {
  margin: 0;
  font-size: 1.35rem;
}

.detail-card__meta {
  display: grid;
  gap: 12px;
  margin: 0;
}

.detail-card__meta div {
  padding: 14px 16px;
  border-radius: 18px;
  background: rgba(20, 78, 110, 0.05);
}

.detail-card__meta dt {
  color: #6d7f89;
  font-size: 0.84rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.detail-card__meta dd {
  margin: 8px 0 0;
  word-break: break-word;
}

.detail-card__actions {
  display: flex;
  gap: 12px;
}

.detail-card__actions button {
  flex: 1;
  padding: 12px 14px;
  border: none;
  border-radius: 14px;
  background: #164d65;
  color: #f8fdff;
  font-weight: 700;
  cursor: pointer;
}

.detail-card__actions .detail-card__ghost {
  background: rgba(20, 78, 110, 0.08);
  color: #164d65;
}

.detail-card__hint {
  margin: 0;
  color: #5b7482;
  line-height: 1.8;
}

.modal {
  position: fixed;
  inset: 0;
  display: grid;
  place-items: center;
  background: rgba(15, 36, 54, 0.28);
  backdrop-filter: blur(8px);
  z-index: 5;
}

.modal__card {
  width: min(560px, calc(100vw - 40px));
  padding: 24px;
  border-radius: 28px;
  background: #fbfdfd;
  box-shadow: 0 28px 70px rgba(18, 50, 71, 0.18);
}

.modal__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
}

.modal__header h3 {
  margin: 8px 0 0;
  font-size: 1.4rem;
}

.modal__close,
.modal__actions button {
  border: none;
  cursor: pointer;
}

.modal__close {
  padding: 10px 12px;
  border-radius: 12px;
  background: rgba(20, 78, 110, 0.08);
  color: #164d65;
}

.modal__label {
  display: block;
  margin-top: 20px;
  margin-bottom: 10px;
  color: #5b7482;
  font-size: 0.86rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.modal__input {
  width: 100%;
  padding: 14px 16px;
  border: 1px solid rgba(18, 50, 71, 0.14);
  border-radius: 16px;
  background: #f7fafb;
}

.modal__actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 18px;
}

.modal__actions button {
  padding: 12px 16px;
  border-radius: 14px;
  background: rgba(20, 78, 110, 0.08);
  color: #164d65;
  font-weight: 700;
}

.modal__actions .modal__primary {
  background: #164d65;
  color: #f8fdff;
}

@media (max-width: 1180px) {
  .app,
  .browser-layout,
  .empty-stage {
    grid-template-columns: 1fr;
  }

  .workspace__hero {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
