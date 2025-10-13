import { test, expect } from '@playwright/test';

// Sets an authenticated user in localStorage before the app loads
async function primeAuth(page: any) {
  await page.addInitScript(() => {
    localStorage.setItem('kmm_user', JSON.stringify({
      username: 'e2e', email: 'e2e@example.com', firstName: 'E2E', lastName: 'Test'
    }));
    localStorage.setItem('kmm_token', 'dummy');
  });
}

test.describe('Brand navigation', () => {
  test('clicking the brand logo navigates to /home', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:4200';
    await primeAuth(page);
    await page.goto(url + '/matches');
    await page.click('.topbar .brand .brand-link');
    await page.waitForURL('**/home');
    expect(page.url()).toContain('/home');
  });

  test('clicking the brand title navigates to /home', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:4200';
    await primeAuth(page);
    await page.goto(url + '/chicago-events');
    await page.click('.topbar .brand .brand-text .brand-link');
    await page.waitForURL('**/home');
    expect(page.url()).toContain('/home');
  });
});
