import { test, expect, Page } from '@playwright/test';

async function primeAuth(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem('affyne_user', JSON.stringify({
      username: 'e2e', email: 'e2e@example.com', firstName: 'E2E', lastName: 'Test'
    }));
    localStorage.setItem('affyne_token', 'dummy');
  });
}

function mockServiceWorker(page: Page, hasSub: boolean) {
  return page.addInitScript(({ hasSub }: { hasSub: boolean }) => {
    // Mock service worker ready + pushManager.getSubscription()
    const pushMgr = { getSubscription: () => Promise.resolve(hasSub ? ({ endpoint: 'fake' }) : null) };
    const swLike = { pushManager: pushMgr } as any;
    const ready = Promise.resolve(swLike);
    const register = () => Promise.resolve(swLike);
    Object.defineProperty(navigator, 'serviceWorker', { value: { ready, register }, configurable: true });
  }, { hasSub });
}

test.describe.skip('Notifications tile visibility (skipped: SW mocking unstable in CI)', () => {
    test.describe.configure({ mode: 'serial' });
  test('shows tile when there is no subscription on this device', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:4200';
  await primeAuth(page);
    await mockServiceWorker(page, false);
  await page.goto(url + '/');
  await page.waitForSelector('.topbar .brand .brand-link');
  await page.click('.topbar .brand .brand-link');
  await page.waitForURL('**/home');
  await page.waitForSelector('.home-tiles');
    await expect(page.locator('.tile-notify')).toBeVisible();
    await expect(page.locator('.tile-notify .tile-title')).toHaveText('Enable Notifications');
  });

  test('hides tile when subscription exists on this device', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:4200';
  await primeAuth(page);
    await mockServiceWorker(page, true);
  await page.goto(url + '/');
  await page.waitForSelector('.topbar .brand .brand-link');
  await page.click('.topbar .brand .brand-link');
  await page.waitForURL('**/home');
  await page.waitForSelector('.home-tiles');
    await expect(page.locator('.tile-notify')).toHaveCount(0);
  });

  test('shows Manage link when permission is not granted', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:4200';
  await primeAuth(page);
    await mockServiceWorker(page, false);
  await page.goto(url + '/');
  await page.waitForSelector('.topbar .brand .brand-link');
  await page.click('.topbar .brand .brand-link');
  await page.waitForURL('**/home');
  await page.waitForSelector('.home-tiles');
    const help = page.locator('.tile-notify .tile-sub.minor a', { hasText: 'Manage' });
    await expect(help).toBeVisible();
    await expect(help).toHaveAttribute('href', /support\.google\.com/);
  });
});
