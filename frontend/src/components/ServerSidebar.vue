<template>
  <aside class="sidebar">
    <div class="brand">
      <span class="brand__eyebrow">Desktop Transfer Deck</span>
      <h1>file-transfer-client</h1>
      <p>连上服务端，浏览远程目录，把文件稳稳拉回本地。</p>
    </div>

    <form class="connect-card" @submit.prevent="emit('connect')">
      <label class="connect-card__label" for="server-url">Server URL</label>
      <input
        id="server-url"
        class="connect-card__input"
        :value="serverInput"
        placeholder="http://127.0.0.1:8080"
        spellcheck="false"
        @input="emit('update:serverInput', ($event.target as HTMLInputElement).value)"
      />
      <button class="connect-card__button" type="submit">
        {{ isConnecting ? 'Connecting...' : connectedServer ? 'Reconnect' : 'Connect' }}
      </button>
    </form>

    <section class="status-card">
      <div class="status-card__row">
        <span class="status-card__dot" :class="{ 'status-card__dot--live': Boolean(connectedServer) }"></span>
        <strong>{{ connectedServer ? 'Connected' : 'Idle' }}</strong>
      </div>
      <p class="status-card__text">{{ connectedServer || 'No active server connection yet.' }}</p>
      <p v-if="checkedAt" class="status-card__meta">Last check: {{ checkedAt }}</p>
    </section>

    <section class="storage-card">
      <div>
        <span class="storage-card__label">Default folder</span>
        <p class="storage-card__value" :title="defaultDownloadDir">{{ defaultDownloadDir || 'Choose a folder' }}</p>
      </div>
      <button class="storage-card__button" type="button" @click="emit('openSettings')">Settings</button>
    </section>

    <section class="recent-card">
      <header class="recent-card__header">
        <h2>Recent servers</h2>
        <span>{{ recentServers.length }}</span>
      </header>
      <div v-if="recentServers.length" class="recent-card__list">
        <button
          v-for="server in recentServers"
          :key="server"
          class="recent-card__item"
          type="button"
          @click="emit('useRecent', server)"
        >
          <span class="recent-card__item-title">{{ server }}</span>
          <span class="recent-card__item-subtitle">Reconnect and browse</span>
        </button>
      </div>
      <p v-else class="recent-card__empty">连接成功过的服务端会出现在这里。</p>
    </section>
  </aside>
</template>

<script setup lang="ts">
defineProps<{
  serverInput: string
  recentServers: string[]
  connectedServer: string
  defaultDownloadDir: string
  isConnecting: boolean
  checkedAt: string
}>()

const emit = defineEmits<{
  (event: 'update:serverInput', value: string): void
  (event: 'connect'): void
  (event: 'useRecent', value: string): void
  (event: 'openSettings'): void
}>()
</script>

<style scoped>
.sidebar {
  display: grid;
  gap: 20px;
  padding: 28px;
  align-self: stretch;
  align-content: start;
  color: #f5fbfc;
  background:
    radial-gradient(circle at top left, rgba(111, 217, 205, 0.28), transparent 38%),
    linear-gradient(180deg, #113348 0%, #0f2436 100%);
  box-shadow: inset -1px 0 0 rgba(255, 255, 255, 0.06);
}

.brand h1 {
  margin: 8px 0 10px;
  font-size: 1.9rem;
  line-height: 1.1;
}

.brand p {
  margin: 0;
  color: rgba(230, 244, 247, 0.8);
  line-height: 1.7;
}

.brand__eyebrow {
  display: inline-flex;
  padding: 6px 10px;
  border-radius: 999px;
  font-size: 0.75rem;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  background: rgba(255, 255, 255, 0.08);
}

.connect-card,
.status-card,
.storage-card,
.recent-card {
  padding: 18px;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(14px);
}

.connect-card__label,
.storage-card__label {
  display: block;
  margin-bottom: 10px;
  color: rgba(228, 244, 247, 0.72);
  font-size: 0.82rem;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.connect-card__input {
  width: 100%;
  padding: 14px 16px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 16px;
  background: rgba(9, 24, 37, 0.55);
  color: #f8fdff;
}

.connect-card__input::placeholder {
  color: rgba(224, 241, 246, 0.48);
}

.connect-card__button,
.storage-card__button {
  margin-top: 12px;
  width: 100%;
  padding: 13px 16px;
  border: none;
  border-radius: 16px;
  background: linear-gradient(135deg, #7ce0d5 0%, #3cb8a5 100%);
  color: #0f2436;
  font-weight: 700;
  cursor: pointer;
}

.status-card__row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.status-card__dot {
  width: 11px;
  height: 11px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.24);
}

.status-card__dot--live {
  background: #7ce0d5;
  box-shadow: 0 0 0 6px rgba(124, 224, 213, 0.12);
}

.status-card__text,
.status-card__meta,
.recent-card__empty {
  margin: 10px 0 0;
  color: rgba(231, 245, 247, 0.74);
  line-height: 1.6;
}

.status-card__meta {
  font-size: 0.9rem;
}

.storage-card {
  display: grid;
  gap: 14px;
}

.storage-card__value {
  margin: 0;
  color: #f8fdff;
  line-height: 1.6;
  word-break: break-word;
}

.recent-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.recent-card__header h2 {
  margin: 0;
  font-size: 1rem;
}

.recent-card__header span {
  display: inline-flex;
  min-width: 28px;
  justify-content: center;
  padding: 4px 8px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.1);
}

.recent-card__list {
  display: grid;
  gap: 10px;
}

.recent-card__item {
  display: grid;
  gap: 4px;
  padding: 12px 14px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  background: rgba(6, 20, 31, 0.26);
  text-align: left;
  color: inherit;
  cursor: pointer;
}

.recent-card__item-title {
  font-weight: 600;
  word-break: break-word;
}

.recent-card__item-subtitle {
  color: rgba(231, 245, 247, 0.62);
  font-size: 0.9rem;
}
</style>
