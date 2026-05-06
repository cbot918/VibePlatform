# GitHub OAuth 登入功能

## 功能說明

使用者透過 GitHub 帳號進行註冊與登入。登入成功後取得包含用戶資訊（頭像、名稱、email）的 session，並在前端頁面顯示登入狀態。

## 流程

```
1. 使用者點擊「Login with GitHub」
2. 前端導向 backend /auth/github
3. Backend 產生隨機 state，寫入 cookie，並 redirect 到 GitHub OAuth 授權頁
4. 使用者在 GitHub 授權
5. GitHub 帶著 code + state redirect 回 /auth/github/callback
6. Backend 驗證 state cookie，用 code 換取 access token
7. 用 access token 取得 GitHub 用戶資料
8. 將用戶存入 store（Upsert），建立 JWT session cookie
9. Redirect 到前端首頁，前端呼叫 /auth/me 取得用戶資料並顯示
```

## 技術

### Backend（Go）

| 項目 | 說明 |
|------|------|
| `github.com/go-chi/chi/v5` | HTTP router |
| `github.com/go-chi/cors` | CORS middleware，允許前端跨域呼叫 |
| `golang.org/x/oauth2` | GitHub OAuth2 流程（code exchange） |
| `github.com/golang-jwt/jwt/v5` | Session token（JWT，7 天有效） |
| Cookie `SameSite=None; Secure` | 支援前後端不同 domain 的 cookie 傳遞 |
| In-memory user store | 以 GitHub ID 做 Upsert，目前無持久化 |

### Frontend（Vue.js + Vite）

| 項目 | 說明 |
|------|------|
| Vue 3 `<script setup>` | 組合式 API，管理登入狀態 |
| `VITE_BACKEND_URL` | 環境變數指定 backend 位置 |
| `credentials: 'include'` | fetch 時帶上跨域 cookie |

### 本機開發環境

| 項目 | 說明 |
|------|------|
| ngrok → backend:3001 | 將本機 backend 暴露為公開 HTTPS URL，供 GitHub callback 使用 |
| frontend:5173 | Vite dev server，直接呼叫 ngrok backend URL |

## API

| 方法 | 路徑 | 說明 |
|------|------|------|
| GET | `/auth/github` | 產生 state，redirect 到 GitHub |
| GET | `/auth/github/callback` | 驗證 state，換 token，建立 session |
| GET | `/auth/me` | 回傳當前登入用戶（需 session cookie） |
| POST | `/auth/logout` | 清除 session cookie |

## 環境變數

```env
GITHUB_CLIENT_ID=       # GitHub OAuth App Client ID
GITHUB_CLIENT_SECRET=   # GitHub OAuth App Client Secret
JWT_SECRET=             # JWT 簽名金鑰（32 字元以上）
PORT=3001               # Backend 監聽 port
FRONTEND_URL=           # 登入成功後跳轉的前端網址
BASE_URL=               # Backend 對外 URL（ngrok 或正式站 domain）
```

## GitHub OAuth App 設定

- Homepage URL：`http://localhost:3001`
- Callback URL：`https://<ngrok 或正式站>/auth/github/callback`

## E2E 測試

使用 Playwright 對所有 auth 端點進行自動化測試：

```bash
cd frontend && npx playwright test tests/auth.spec.js
```

測試項目：
- 未登入時首頁顯示登入按鈕
- 點擊登入按鈕跳轉到 GitHub
- `/auth/me` 未登入回 401
- `/auth/logout` 回 200
- `/auth/github/callback` 無 state 回 400
- `/auth/github` redirect 帶正確 client_id
