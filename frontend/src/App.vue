<template>
  <div class="page">
    <!-- Loading -->
    <div v-if="status === 'loading'" class="card center">
      <div class="spinner"></div>
      <p class="hint">載入中...</p>
    </div>

    <!-- 未登入 -->
    <div v-else-if="status === 'loggedOut'" class="card center">
      <h1>VibePlatform</h1>
      <p class="subtitle">使用 GitHub 帳號登入後開始 Vibe Coding</p>
      <a :href="backendURL + '/auth/github'" class="btn-github">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="white">
          <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z"/>
        </svg>
        Login with GitHub
      </a>
    </div>

    <!-- 已登入 -->
    <template v-else>
      <!-- Topbar -->
      <header class="topbar">
        <span class="brand">VibePlatform</span>
        <nav class="nav">
          <button :class="['nav-btn', view === 'projects' && 'active']" @click="view = 'projects'">Projects</button>
          <button :class="['nav-btn', view === 'settings' && 'active']" @click="view = 'settings'">Settings</button>
        </nav>
        <div class="topbar-user">
          <img class="topbar-avatar" :src="user.avatar_url" />
          <span class="topbar-login">@{{ user.login }}</span>
          <button class="btn-sm" @click="logout">登出</button>
        </div>
      </header>

      <!-- Settings View -->
      <main v-if="view === 'settings'" class="content">
        <div class="card wide">
          <h2>Settings</h2>
          <p class="hint">設定你的 Anthropic API Key，用於 Claude Code</p>
          <div class="field">
            <label>Anthropic API Key</label>
            <div class="key-row">
              <span v-if="settings.has_key" class="masked-key">{{ settings.masked_key }}</span>
              <span v-else class="hint">尚未設定</span>
            </div>
            <input v-model="newApiKey" type="password" placeholder="sk-ant-..." class="input" />
            <button class="btn-primary" :disabled="!newApiKey || saving" @click="saveSettings">
              {{ saving ? '儲存中...' : '儲存 API Key' }}
            </button>
            <p v-if="saveMsg" class="save-msg">{{ saveMsg }}</p>
          </div>
        </div>
      </main>

      <!-- Projects View -->
      <main v-else class="content">
        <div class="projects-header">
          <h2>My Projects</h2>
          <button class="btn-primary" @click="showCreate = true">+ 建立 Project</button>
        </div>

        <!-- Create form -->
        <div v-if="showCreate" class="card create-card">
          <h3>建立新 Project</h3>
          <p class="hint">名稱只能使用小寫英數字和連字號（例如：my-project）</p>
          <div v-if="!settings.has_key" class="warn-box">
            請先到 <button class="link-btn" @click="view = 'settings'">Settings</button> 設定 Anthropic API Key
          </div>
          <input v-model="newProjectName" :disabled="!settings.has_key" type="text" placeholder="my-project" class="input" @keydown.enter="createProject" />
          <div class="btn-row">
            <button class="btn-outline" @click="showCreate = false">取消</button>
            <button class="btn-primary" :disabled="!newProjectName || !settings.has_key || creating" @click="createProject">
              {{ creating ? '建立中...' : '建立並啟動' }}
            </button>
          </div>
          <p v-if="createError" class="error-msg">{{ createError }}</p>
        </div>

        <!-- Project list -->
        <div v-if="projects.length === 0 && !showCreate" class="empty-state">
          <p>還沒有 Project，點上方「建立 Project」開始</p>
        </div>
        <div v-for="p in projects" :key="p.name" class="project-card">
          <div class="project-left">
            <span class="project-name">{{ p.name }}</span>
            <span class="status-badge" :class="p.status">{{ p.status }}</span>
          </div>
          <div class="project-actions">
            <a v-if="p.status === 'running'" :href="'http://localhost:' + p.host_port + '/?folder=/home/coder/' + p.name" target="_blank" class="btn-open">開啟 Workspace</a>
            <button class="btn-sm danger" @click="stopProject(p.name)" :disabled="p.status !== 'running'">停止</button>
            <button class="btn-sm" @click="deleteProject(p.name)">刪除</button>
          </div>
        </div>
      </main>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'

const backendURL = import.meta.env.VITE_BACKEND_URL || ''

const status = ref('loading')
const user = ref(null)
const view = ref('projects')

const settings = ref({ has_key: false, masked_key: '' })
const newApiKey = ref('')
const saving = ref(false)
const saveMsg = ref('')

const projectsMap = ref({})
const projects = computed(() => Object.values(projectsMap.value))
const showCreate = ref(false)
const newProjectName = ref('')
const creating = ref(false)
const createError = ref('')

async function api(path, options = {}) {
  return fetch(backendURL + path, { credentials: 'include', ...options })
}

async function checkAuth() {
  const res = await api('/auth/me')
  if (res.ok) {
    user.value = await res.json()
    status.value = 'loggedIn'
    await Promise.all([fetchSettings(), fetchProjects()])
  } else {
    status.value = 'loggedOut'
  }
}

async function fetchSettings() {
  const res = await api('/user/settings')
  if (res.ok) settings.value = await res.json()
}

async function saveSettings() {
  saving.value = true
  saveMsg.value = ''
  const res = await api('/user/settings', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ anthropic_api_key: newApiKey.value }),
  })
  if (res.ok) {
    saveMsg.value = '已儲存'
    newApiKey.value = ''
    await fetchSettings()
  } else {
    saveMsg.value = '儲存失敗'
  }
  saving.value = false
}

async function fetchProjects() {
  const res = await api('/project')
  if (res.ok) {
    const list = await res.json()
    const map = {}
    ;(list || []).forEach(p => { map[p.name] = p })
    projectsMap.value = map
  }
}

async function createProject() {
  if (!newProjectName.value || creating.value) return
  creating.value = true
  createError.value = ''
  const res = await api('/project', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: newProjectName.value }),
  })
  if (res.ok) {
    const info = await res.json()
    projectsMap.value[info.name] = info
    newProjectName.value = ''
    showCreate.value = false
  } else {
    createError.value = await res.text()
  }
  creating.value = false
}

async function stopProject(name) {
  await api(`/project/${name}/stop`, { method: 'POST' })
  await fetchProjects()
}

async function deleteProject(name) {
  await api(`/project/${name}`, { method: 'DELETE' })
  delete projectsMap.value[name]
  projectsMap.value = { ...projectsMap.value }
}

async function logout() {
  await api('/auth/logout', { method: 'POST' })
  user.value = null
  status.value = 'loggedOut'
}

onMounted(checkAuth)
</script>

<style scoped>
* { box-sizing: border-box; }

.page {
  min-height: 100vh;
  background: #f5f5f5;
  font-family: system-ui, -apple-system, sans-serif;
}

/* Center card (login / loading) */
.page:has(.center) {
  display: flex;
  align-items: center;
  justify-content: center;
}
.card.center {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
  padding: 40px;
  width: 360px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}
h1 { font-size: 22px; color: #111; }
.subtitle { color: #888; font-size: 13px; }

/* Topbar */
.topbar {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 0 24px;
  height: 52px;
  background: white;
  border-bottom: 1px solid #e8e8e8;
}
.brand { font-weight: 700; font-size: 16px; color: #111; margin-right: 8px; }
.nav { display: flex; gap: 4px; flex: 1; }
.nav-btn {
  padding: 6px 14px;
  border: none;
  background: none;
  border-radius: 6px;
  font-size: 14px;
  color: #555;
  cursor: pointer;
}
.nav-btn.active { background: #f0f0f0; color: #111; font-weight: 500; }
.nav-btn:hover:not(.active) { background: #f8f8f8; }
.topbar-user { display: flex; align-items: center; gap: 8px; }
.topbar-avatar { width: 28px; height: 28px; border-radius: 50%; }
.topbar-login { font-size: 13px; color: #555; }

/* Content */
.content { max-width: 760px; margin: 32px auto; padding: 0 24px; }

.card { background: white; border-radius: 10px; box-shadow: 0 1px 4px rgba(0,0,0,0.07); padding: 28px; }
.card.wide { max-width: 520px; }
h2 { font-size: 18px; color: #111; margin-bottom: 4px; }
h3 { font-size: 15px; color: #111; margin-bottom: 8px; }

/* Settings */
.field { display: flex; flex-direction: column; gap: 10px; margin-top: 16px; }
label { font-size: 13px; font-weight: 500; color: #555; }
.key-row { font-size: 13px; color: #888; }
.masked-key { font-family: monospace; color: #333; }
.input {
  padding: 9px 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 14px;
  outline: none;
}
.input:focus { border-color: #2da44e; }
.save-msg { font-size: 13px; color: #2da44e; }

/* Projects */
.projects-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.create-card { margin-bottom: 16px; display: flex; flex-direction: column; gap: 10px; }
.warn-box {
  background: #fff8e6;
  border: 1px solid #f0d080;
  border-radius: 6px;
  padding: 10px 14px;
  font-size: 13px;
  color: #7a5c00;
}
.error-msg { font-size: 13px; color: #e05c5c; }
.empty-state { text-align: center; color: #aaa; font-size: 14px; padding: 48px 0; }

.project-card {
  background: white;
  border-radius: 10px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.07);
  padding: 16px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}
.project-left { display: flex; align-items: center; gap: 10px; }
.project-name { font-weight: 600; font-size: 15px; color: #111; }
.project-actions { display: flex; align-items: center; gap: 8px; }

/* Status badges */
.status-badge { font-size: 11px; padding: 2px 8px; border-radius: 20px; font-weight: 500; }
.status-badge.running { background: #e6ffed; color: #2da44e; }
.status-badge.stopped { background: #f0f0f0; color: #888; }
.status-badge.exited  { background: #fff0f0; color: #e05c5c; }

/* Buttons */
.btn-github {
  display: flex; align-items: center; gap: 10px;
  width: 100%; padding: 12px; background: #24292f; color: white;
  border: none; border-radius: 8px; font-size: 14px; font-weight: 500;
  cursor: pointer; text-decoration: none; justify-content: center;
  transition: opacity 0.15s;
}
.btn-github:hover { opacity: 0.85; }
.btn-primary {
  padding: 9px 18px; background: #2da44e; color: white;
  border: none; border-radius: 6px; font-size: 14px; font-weight: 500;
  cursor: pointer; transition: opacity 0.15s; white-space: nowrap;
}
.btn-primary:hover:not(:disabled) { opacity: 0.85; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-outline {
  padding: 9px 18px; background: white; color: #555;
  border: 1px solid #ddd; border-radius: 6px; font-size: 14px;
  cursor: pointer; transition: opacity 0.15s;
}
.btn-outline:hover { background: #f5f5f5; }
.btn-open {
  padding: 7px 14px; background: #0969da; color: white;
  border-radius: 6px; font-size: 13px; font-weight: 500;
  text-decoration: none; transition: opacity 0.15s; white-space: nowrap;
}
.btn-open:hover { opacity: 0.85; }
.btn-sm {
  padding: 6px 12px; background: #f5f5f5; color: #555;
  border: 1px solid #e0e0e0; border-radius: 6px; font-size: 13px;
  cursor: pointer; transition: opacity 0.15s;
}
.btn-sm:hover:not(:disabled) { background: #ebebeb; }
.btn-sm:disabled { opacity: 0.4; cursor: not-allowed; }
.btn-sm.danger { color: #e05c5c; border-color: #f5c6c6; }
.btn-sm.danger:hover { background: #fff0f0; }
.link-btn { background: none; border: none; color: #0969da; cursor: pointer; font-size: inherit; padding: 0; text-decoration: underline; }

.btn-row { display: flex; gap: 8px; }

/* Spinner */
.spinner {
  width: 32px; height: 32px;
  border: 3px solid #e0e0e0; border-top-color: #24292f;
  border-radius: 50%; animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
.hint { color: #aaa; font-size: 13px; }
</style>
