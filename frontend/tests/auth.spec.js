// @ts-check
import { test, expect } from '@playwright/test';

test.describe('未登入狀態', () => {
  test('首頁顯示登入按鈕', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText('Login with GitHub')).toBeVisible();
    await expect(page.getByText('VibePlatform')).toBeVisible();
  });

  test('點登入按鈕跳轉到 GitHub OAuth 頁', async ({ page }) => {
    await page.goto('/');
    await page.getByText('Login with GitHub').click();

    // 等待跳轉到 GitHub（未登入時會先到 github.com/login，已登入會到 oauth/authorize）
    await page.waitForURL(/github\.com/, { timeout: 8000 });

    const url = page.url();
    expect(url).toContain('github.com');
    // client_id 帶在 URL 或 return_to 參數中
    expect(url).toContain('client_id=Ov23liLzy6p9SzZHVIXJ');
  });
});

test.describe('API 端點', () => {
  test('/auth/me 未登入回傳 401', async ({ request }) => {
    const res = await request.get('/auth/me');
    expect(res.status()).toBe(401);
  });

  test('/auth/logout 回傳 200', async ({ request }) => {
    const res = await request.post('/auth/logout');
    expect(res.status()).toBe(200);
  });

  test('/auth/github/callback 無 state 回傳 400', async ({ request }) => {
    const res = await request.get('/auth/github/callback');
    expect(res.status()).toBe(400);
  });

  test('/auth/github redirect 帶正確 client_id', async ({ request }) => {
    const res = await request.get('/auth/github', { maxRedirects: 0 });
    expect(res.status()).toBe(302);
    const location = res.headers()['location'];
    expect(location).toContain('github.com/login/oauth/authorize');
    expect(location).toContain('client_id=Ov23liLzy6p9SzZHVIXJ');
  });
});
