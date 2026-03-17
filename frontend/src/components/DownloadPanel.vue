<template>
  <section class="downloads">
    <header class="downloads__header">
      <div>
        <span class="downloads__eyebrow">Task board</span>
        <h2>Downloads</h2>
      </div>
      <strong>{{ runningCount }} running</strong>
    </header>

    <div v-if="!orderedTasks.length" class="downloads__empty">
      队列还空着。下载文件后，进度和结果会显示在这里。
    </div>

    <div v-else class="downloads__list">
      <article v-for="task in orderedTasks" :key="task.id" class="downloads__item">
        <div class="downloads__item-top">
          <div>
            <strong>{{ taskName(task.remote_path) }}</strong>
            <p>{{ task.remote_path }}</p>
          </div>
          <span class="downloads__state" :class="`downloads__state--${task.state}`">
            {{ stateLabel(task.state) }}
          </span>
        </div>

        <div class="downloads__meter">
          <div class="downloads__meter-fill" :style="{ width: progress(task) }"></div>
        </div>

        <div class="downloads__meta">
          <span>{{ formatSize(task.written_bytes) }} / {{ task.total_bytes > 0 ? formatSize(task.total_bytes) : 'Unknown' }}</span>
          <span v-if="task.finished_at">{{ formatDate(task.finished_at) }}</span>
          <span v-else-if="task.started_at">{{ formatDate(task.started_at) }}</span>
        </div>

        <p v-if="task.error_message && task.state === 'failed'" class="downloads__error">{{ task.error_message }}</p>

        <div class="downloads__actions">
          <button
            v-if="task.state === 'running' || task.state === 'queued'"
            type="button"
            @click="emit('cancel', task)"
          >
            Cancel
          </button>
          <button
            v-if="task.state === 'done'"
            type="button"
            @click="emit('openFolder', task)"
          >
            Open Folder
          </button>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { DownloadState, DownloadTask } from '../types'

const props = defineProps<{
  tasks: DownloadTask[]
}>()

const emit = defineEmits<{
  (event: 'cancel', value: DownloadTask): void
  (event: 'openFolder', value: DownloadTask): void
}>()

const orderedTasks = computed(() => {
  return [...props.tasks].sort((left, right) => {
    const leftTime = Date.parse(left.finished_at || left.started_at || '') || 0
    const rightTime = Date.parse(right.finished_at || right.started_at || '') || 0
    return rightTime - leftTime
  })
})

const runningCount = computed(() => props.tasks.filter((task) => task.state === 'running').length)

function taskName(remotePath: string): string {
  const clean = remotePath.replace(/\/+$/, '')
  const parts = clean.split('/')
  return parts.at(-1) || remotePath
}

function progress(task: DownloadTask): string {
  if (!task.total_bytes || task.total_bytes <= 0) {
    return task.state === 'done' ? '100%' : '18%'
  }
  return `${Math.min((task.written_bytes / task.total_bytes) * 100, 100)}%`
}

function formatSize(size: number): string {
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

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

function stateLabel(state: DownloadState): string {
  switch (state) {
    case 'queued':
      return 'Queued'
    case 'running':
      return 'Running'
    case 'done':
      return 'Done'
    case 'failed':
      return 'Failed'
    case 'canceled':
      return 'Canceled'
    default:
      return state
  }
}
</script>

<style scoped>
.downloads {
  display: grid;
  gap: 16px;
}

.downloads__header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
}

.downloads__eyebrow {
  display: inline-flex;
  margin-bottom: 8px;
  color: #6d7f89;
  font-size: 0.76rem;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.downloads__header h2 {
  margin: 0;
  font-size: 1.15rem;
}

.downloads__header strong {
  color: #20516c;
}

.downloads__empty {
  padding: 18px;
  border-radius: 20px;
  background: rgba(20, 78, 110, 0.06);
  color: #5c7682;
  line-height: 1.7;
}

.downloads__list {
  display: grid;
  gap: 14px;
}

.downloads__item {
  padding: 16px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(18, 50, 71, 0.08);
}

.downloads__item-top {
  display: flex;
  gap: 14px;
  justify-content: space-between;
}

.downloads__item-top p {
  margin: 6px 0 0;
  color: #6d7f89;
  font-size: 0.9rem;
  word-break: break-word;
}

.downloads__state {
  align-self: flex-start;
  padding: 6px 10px;
  border-radius: 999px;
  font-size: 0.76rem;
  font-weight: 700;
  text-transform: uppercase;
}

.downloads__state--queued {
  background: #eef1f4;
  color: #54616c;
}

.downloads__state--running {
  background: #dff3f0;
  color: #156a6f;
}

.downloads__state--done {
  background: #d8efe0;
  color: #2f7e53;
}

.downloads__state--failed,
.downloads__state--canceled {
  background: #f8e2e2;
  color: #a14d4d;
}

.downloads__meter {
  margin-top: 14px;
  height: 10px;
  border-radius: 999px;
  overflow: hidden;
  background: rgba(20, 78, 110, 0.08);
}

.downloads__meter-fill {
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #1b7f83 0%, #5ec7bd 100%);
}

.downloads__meta {
  display: flex;
  justify-content: space-between;
  margin-top: 10px;
  color: #6d7f89;
  font-size: 0.9rem;
}

.downloads__error {
  margin: 10px 0 0;
  color: #a14d4d;
  line-height: 1.6;
}

.downloads__actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}

.downloads__actions button {
  padding: 10px 14px;
  border: none;
  border-radius: 12px;
  background: rgba(20, 78, 110, 0.08);
  color: #164d65;
  font-weight: 700;
  cursor: pointer;
}
</style>
