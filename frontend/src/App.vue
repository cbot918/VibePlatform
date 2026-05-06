<template>
  <div class="container">
    <!-- Loading -->
    <div v-if="status === 'loading'" class="card">
      <div class="spinner"></div>
      <p class="hint">載入中...</p>
    </div>

    <!-- 已登入 -->
    <div v-else-if="status === 'loggedIn'" class="card">
      <img class="avatar" :src="user.avatar_url" alt="avatar" />
      <div class="name">{{ user.name || user.login }}</div>
      <div class="login">@{{ user.login }}</div>
      <div v-if="user.email" class="email">{{ user.email }}</div>
      <div class="badge">已登入 GitHub</div>
      <button class="btn-logout" @click="logout">登出</button>
    </div>

    <!-- 未登入 -->
    <div v-else class="card">
      <h1>VibePlatform</h1>
      <p class="subtitle">使用 GitHub 帳號登入</p>
      <a :href="backendURL + '/auth/github'" class="btn-github">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="white">
          <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z"/>
        </svg>
        Login with GitHub
      </a>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const backendURL = import.meta.env.VITE_BACKEND_URL || ''

const status = ref('loading')
const user = ref(null)

async function checkAuth() {
  try {
    const res = await fetch(backendURL + '/auth/me', { credentials: 'include' })
    if (res.ok) {
      user.value = await res.json()
      status.value = 'loggedIn'
    } else {
      status.value = 'loggedOut'
    }
  } catch {
    status.value = 'loggedOut'
  }
}

async function logout() {
  await fetch(backendURL + '/auth/logout', { method: 'POST', credentials: 'include' })
  user.value = null
  status.value = 'loggedOut'
}

onMounted(checkAuth)
</script>

<style scoped>
.container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f5f5;
}

.card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
  padding: 40px;
  width: 360px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

h1 { font-size: 20px; color: #111; margin-bottom: 4px; }
.subtitle { color: #888; font-size: 13px; margin-bottom: 16px; }

.avatar { width: 72px; height: 72px; border-radius: 50%; }
.name { font-size: 18px; font-weight: 600; color: #111; }
.login { color: #888; font-size: 13px; }
.email { color: #555; font-size: 13px; }

.badge {
  background: #e6ffed;
  color: #2da44e;
  font-size: 12px;
  padding: 2px 10px;
  border-radius: 20px;
  margin: 8px 0;
}

.btn-github {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 12px;
  background: #24292f;
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  text-decoration: none;
  justify-content: center;
  margin-top: 8px;
  transition: opacity 0.15s;
}
.btn-github:hover { opacity: 0.85; }

.btn-logout {
  width: 100%;
  padding: 12px;
  background: #f5f5f5;
  color: #555;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s;
  margin-top: 8px;
}
.btn-logout:hover { opacity: 0.85; }

.spinner {
  width: 32px; height: 32px;
  border: 3px solid #e0e0e0;
  border-top-color: #24292f;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.hint { color: #aaa; font-size: 13px; }
</style>
