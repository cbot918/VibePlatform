# Reset 功能

## 目的

讓手動測試更方便：一鍵停止 container 並清空所有儲存資料，回到乾淨狀態。

## 已完成

### 後端

- `internal/store/container.go` — 新增 `Clear()`，重設 in-memory map 並寫回空 JSON
- `internal/store/project.go` — 新增 `Clear()`，同上
- `internal/store/settings.go` — 新增 `Clear()`，同上
- `internal/handler/debug.go` — 新建 `DebugHandler`，實作 `POST /debug/reset`：
  1. 從 containers store 取得 containerID
  2. 呼叫 `docker.Stop()` 刪除 container（失敗只 log）
  3. 依序清空 containers / projects / settings store
- `internal/server/server.go` — 接 `debugHandler`，註冊 `POST /debug/reset` 路由
- `cmd/server/main.go` — 建立 `debugHandler` 並傳入 server
- `internal/server/server_test.go` — 補 `NewDebugHandler(nil...)` 參數

### 前端

- `frontend/src/App.vue` — Settings 頁底部新增 **Tests** section：
  - 紅色「重置環境」按鈕（`btn-danger` style）
  - 按下先 `confirm()` 確認，再 `POST /debug/reset`
  - 成功後自動清空前端 projects / settings 狀態，不須重新整理頁面

## 同批一起進的功能（gh CLI 認證）

- `docker/Dockerfile` — 加入 git + gh CLI（gh 2.92.0）安裝
- `internal/docker/client.go` — 新增 `execIn` helper、`execWithStdin` helper、`ConfigureGit` 方法
  - 每次建立 project 時對 running container 執行 git config / gh auth login
  - 使用 `ContainerExecAttach` 寫 stdin，避免 shell 插值問題
- `internal/handler/project.go` — `CodeServerDocker` 介面加 `ConfigureGit`，`HandleCreate` 在 EnsureCodeServer 後若有 GitToken 則呼叫
- `internal/handler/project_test.go` — mock 補 `configureGitFn`，新增兩個測試
- `internal/handler/settings.go` / `settings_test.go` — UserSettings 加 git 欄位相關處理（前一版）
