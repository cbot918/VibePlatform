# GitHub Settings Feature

## 目的

讓 code-server container 裡的 agent 能 push GitHub，需要在 settings 存 git user、email 和 token，並在啟動 container 時注入為環境變數。

## 已完成

### 後端

- `internal/store/settings.go` — `UserSettings` 新增 `GitUser`, `GitEmail`, `GitToken`
- `GET /user/settings` 多回傳 `git_user`, `git_email`, `has_git_token`, `masked_git_token`
- `POST /user/settings` 改為 patch 語意 — 只更新非空欄位，不覆蓋已存在的其他設定
- `internal/handler/settings_test.go` 新增 4 個測試：git 欄位讀取、儲存、patch 語意

### 前端

- Settings 頁分兩個 section：Anthropic / GitHub
- 載入時優先填入已存的 `git_user` / `git_email`；若尚未設定，自動從 GitHub OAuth 帶入（`user.name` → `user.login`，`user.email`）
- token 欄位每次都要重輸（password field，安全考量）
- 統一一個「儲存設定」按鈕，四個欄位全填才能送出

## 待做

### 1. Docker image 加 git + gh CLI

`docker/Dockerfile` 補裝：

```dockerfile
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
      | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
      | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update && apt-get install -y git gh \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
```

重新 build image：`docker build -t vibeplatform-code-server:latest ./docker`

### 2. Container 啟動時注入環境變數

`internal/handler/project.go` 或 `internal/docker/client.go` 在啟動 code-server container 時，從 settings store 讀取 git 設定並注入：

```
GITHUB_TOKEN=<git_token>
GIT_AUTHOR_NAME=<git_user>
GIT_AUTHOR_EMAIL=<git_email>
GIT_COMMITTER_NAME=<git_user>
GIT_COMMITTER_EMAIL=<git_email>
```

這樣 `gh` CLI 和 git 在 container 內就能自動認證，不需要額外設定。

### 3. OAuth scope（備註）

目前用 PAT（`ghp_...`）最簡單，不需要改 OAuth scope。
若未來想直接用 OAuth token push，需在 `internal/auth/github.go` 加 `repo` scope。
