<template>
  <section class="browser">
    <header class="browser__toolbar">
      <nav class="browser__crumbs" aria-label="Breadcrumb">
        <button
          v-for="crumb in breadcrumbs"
          :key="crumb.path"
          class="browser__crumb"
          type="button"
          @click="emit('navigate', crumb.path)"
        >
          {{ crumb.label }}
        </button>
      </nav>
      <div class="browser__toolbar-actions">
        <input
          :value="filterText"
          class="browser__filter"
          placeholder="Filter current folder"
          spellcheck="false"
          @input="emit('update:filterText', ($event.target as HTMLInputElement).value)"
        />
        <button
          v-if="filterText"
          class="browser__clear"
          type="button"
          @click="emit('update:filterText', '')"
        >
          Clear
        </button>
        <button class="browser__refresh" type="button" @click="emit('refresh')">Refresh</button>
      </div>
    </header>

    <div class="browser__head">
      <button
        class="browser__sort"
        :class="{ 'browser__sort--active': isSortedBy('name') }"
        type="button"
        @click="emit('sort', 'name')"
      >
        <span>Name</span>
        <span class="browser__sort-marker">{{ sortMarker('name') }}</span>
      </button>
      <span>Kind</span>
      <button
        class="browser__sort"
        :class="{ 'browser__sort--active': isSortedBy('size') }"
        type="button"
        @click="emit('sort', 'size')"
      >
        <span>Size</span>
        <span class="browser__sort-marker">{{ sortMarker('size') }}</span>
      </button>
      <button
        class="browser__sort"
        :class="{ 'browser__sort--active': isSortedBy('modified') }"
        type="button"
        @click="emit('sort', 'modified')"
      >
        <span>Modified</span>
        <span class="browser__sort-marker">{{ sortMarker('modified') }}</span>
      </button>
      <span>Action</span>
    </div>

    <div v-if="loading" class="browser__empty">
      <strong>{{ loadingTitle }}</strong>
      <p>{{ loadingHint }}</p>
    </div>

    <div v-else-if="!entries.length" class="browser__empty">
      <strong>{{ emptyTitle }}</strong>
      <p>{{ emptyHint }}</p>
    </div>

    <div v-else class="browser__rows">
      <div
        v-for="entry in entries"
        :key="entry.path"
        class="browser__row"
        :class="{ 'browser__row--selected': entry.path === selectedPath }"
        tabindex="0"
        @click="emit('select', entry)"
        @dblclick="emit('open', entry)"
        @keydown.enter.prevent="emit('open', entry)"
      >
        <div class="browser__name">
          <span class="browser__icon" :class="{ 'browser__icon--dir': entry.is_dir }">
            {{ entry.is_dir ? 'DIR' : 'FILE' }}
          </span>
          <div>
            <strong>{{ entry.name }}</strong>
            <p>{{ entry.path }}</p>
          </div>
        </div>
        <span>{{ kindLabel(entry) }}</span>
        <span>{{ formatSize(entry.size, entry.is_dir) }}</span>
        <span>{{ formatDate(entry.last_modified) }}</span>
        <div class="browser__action-wrap">
          <button class="browser__action" type="button" @click.stop="emit('open', entry)">
            {{ entry.is_dir ? 'Enter' : 'Download' }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Entry, SortKey, SortState } from '../types'

const props = withDefaults(defineProps<{
  entries: Entry[]
  currentPath: string
  loading: boolean
  selectedPath: string
  filterText: string
  sortState: SortState
  loadingTitle?: string
  loadingHint?: string
  emptyTitle?: string
  emptyHint?: string
}>(), {
  loadingTitle: 'Loading directory…',
  loadingHint: '正在从服务端拉取最新目录内容。',
  emptyTitle: 'This folder is empty.',
  emptyHint: '换个路径看看，或者刷新一下当前目录。',
})

const emit = defineEmits<{
  (event: 'select', value: Entry): void
  (event: 'open', value: Entry): void
  (event: 'navigate', value: string): void
  (event: 'refresh'): void
  (event: 'sort', value: SortKey): void
  (event: 'update:filterText', value: string): void
}>()

const breadcrumbs = computed(() => {
  const current = props.currentPath || '/'
  const cleaned = current.replace(/^\/+|\/+$/g, '')
  const segments = cleaned ? cleaned.split('/') : []

  const list = [{ label: 'Root', path: '/' }]
  let cursor = ''
  for (const segment of segments) {
    cursor += `/${segment}`
    list.push({ label: segment, path: cursor })
  }
  return list
})

function formatDate(value: string): string {
  if (!value) {
    return '-'
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

function kindLabel(entry: Entry): string {
  if (entry.is_dir) {
    return 'Folder'
  }

  const parts = entry.name.split('.')
  if (parts.length > 1) {
    return parts.at(-1)?.toUpperCase() || 'File'
  }
  return 'File'
}

function isSortedBy(key: SortKey): boolean {
  return props.sortState.key === key
}

function sortMarker(key: SortKey): string {
  if (!isSortedBy(key)) {
    return ''
  }
  return props.sortState.direction === 'asc' ? '^' : 'v'
}
</script>

<style scoped>
.browser {
  display: grid;
  gap: 14px;
}

.browser__toolbar,
.browser__head,
.browser__row {
  display: grid;
  grid-template-columns: minmax(0, 2.2fr) 0.8fr 0.8fr 1fr 0.8fr;
  gap: 16px;
  align-items: center;
}

.browser__toolbar {
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
}

.browser__toolbar-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
}

.browser__crumbs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.browser__crumb,
.browser__refresh,
.browser__clear,
.browser__action {
  border: none;
  cursor: pointer;
}

.browser__crumb {
  padding: 8px 12px;
  border-radius: 999px;
  background: rgba(20, 78, 110, 0.08);
  color: #20516c;
}

.browser__refresh {
  padding: 10px 16px;
  border-radius: 14px;
  background: #164d65;
  color: #f8fdff;
  font-weight: 600;
}

.browser__filter {
  width: min(260px, 100%);
  min-width: 0;
  padding: 10px 14px;
  border: 1px solid rgba(18, 50, 71, 0.14);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.82);
}

.browser__clear {
  padding: 10px 14px;
  border-radius: 14px;
  background: rgba(20, 78, 110, 0.08);
  color: #164d65;
  font-weight: 600;
}

.browser__head {
  padding: 0 18px;
  color: #607886;
  font-size: 0.88rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.browser__sort {
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  gap: 8px;
  padding: 0;
  border: none;
  background: transparent;
  color: inherit;
  font: inherit;
  text-transform: inherit;
  letter-spacing: inherit;
  cursor: pointer;
}

.browser__sort--active {
  color: #20516c;
}

.browser__sort-marker {
  min-width: 0.75rem;
  font-size: 0.78rem;
}

.browser__rows {
  display: grid;
  gap: 12px;
}

.browser__row {
  padding: 16px 18px;
  border: 1px solid rgba(18, 50, 71, 0.08);
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.75);
  transition: transform 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease;
}

.browser__row:hover,
.browser__row--selected {
  transform: translateY(-1px);
  border-color: rgba(27, 127, 131, 0.22);
  box-shadow: 0 14px 30px rgba(18, 50, 71, 0.08);
}

.browser__name {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}

.browser__name > div {
  min-width: 0;
}

.browser__name strong,
.browser__name p {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.browser__name p {
  margin: 5px 0 0;
  color: #6d7f89;
  font-size: 0.92rem;
}

.browser__icon {
  display: inline-flex;
  width: 56px;
  justify-content: center;
  padding: 8px 10px;
  border-radius: 14px;
  background: rgba(20, 78, 110, 0.08);
  color: #20516c;
  font-size: 0.74rem;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.browser__icon--dir {
  background: rgba(27, 127, 131, 0.14);
  color: #14696e;
}

.browser__action-wrap {
  display: flex;
  justify-content: flex-end;
}

.browser__action {
  padding: 10px 14px;
  border-radius: 12px;
  background: rgba(20, 78, 110, 0.08);
  color: #164d65;
  font-weight: 700;
}

.browser__empty {
  display: grid;
  place-items: center;
  min-height: 280px;
  padding: 28px;
  border-radius: 28px;
  border: 1px dashed rgba(18, 50, 71, 0.16);
  background: rgba(255, 255, 255, 0.54);
  text-align: center;
}

.browser__empty strong {
  font-size: 1.08rem;
}

.browser__empty p {
  margin: 10px 0 0;
  color: #6d7f89;
}

@media (max-width: 920px) {
  .browser__toolbar {
    grid-template-columns: 1fr;
  }

  .browser__toolbar-actions {
    justify-content: flex-start;
  }

  .browser__filter {
    width: 100%;
  }
}
</style>
