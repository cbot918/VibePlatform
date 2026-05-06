  API endpoints:
  - GET /auth/github → 導向 GitHub OAuth
  - GET /auth/github/callback → 交換 code、建立 session cookie、導回前端
  - GET /auth/me → 回傳當前用戶資訊（需 session cookie）
  - POST /auth/logout → 清除 session cookie