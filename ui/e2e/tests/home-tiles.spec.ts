import { test, expect, Page } from '@playwright/test';

async function primeAuth(page: Page, username = 'e2e') {
  await page.addInitScript(({ username }) => {
    localStorage.setItem('kmm_user', JSON.stringify({
      username,
      email: username + '@example.com',
      firstName: 'E2E',
      lastName: 'Test'
    }));
    localStorage.setItem('kmm_token', 'dummy');
    // Ensure per-user spotify readiness is cleared unless explicitly set by the test
    localStorage.removeItem(`kmmSpotifyReady:${username}`);
    localStorage.removeItem(`kmmSpotifyReadyTs:${username}`);
    // Clear visited flags so menu items are hidden initially
    localStorage.removeItem('kmmVisitedMatches');
    localStorage.removeItem('kmmVisitedEvents');
  }, { username });
}

test.describe('Home tiles and menu visibility', () => {
  test('shows Link with Spotify tile for new users and hides menu until visited', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:4200';
    await primeAuth(page, 'newuser');
    await page.goto(url + '/home');

    // The Link with Spotify tile should be visible; Re-sync should not
    await expect(page.getByRole('button', { name: 'Link with Spotify' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Re-sync Spotify' })).toHaveCount(0);

    // Header menu items are hidden initially
    await expect(page.locator('header .desktop-nav a[routerlink="/matches"]')).toHaveCount(0);
    await expect(page.locator('header .desktop-nav a[routerlink="/chicago-events"]')).toHaveCount(0);

    // Clicking a primary tile marks both as visited and navigates
    await page.getByRole('button', { name: 'Open Matches' }).click();
    await page.waitForURL('**/matches');

    // After visiting, the header menu links should be visible
    await expect(page.locator('header .desktop-nav a[routerlink="/matches"]')).toBeVisible();
    await expect(page.locator('header .desktop-nav a[routerlink="/chicago-events"]')).toBeVisible();
  });

  test('shows Re-sync Spotify tile after linking (per-user scoped)', async ({ page, baseURL }) => {
    const url = baseURL ?? 'http://localhost:4200';
    const username = 'linkeduser';
    await page.addInitScript(({ username }) => {
      localStorage.setItem('kmm_user', JSON.stringify({
        username,
        email: username + '@example.com',
        firstName: 'E2E',
        lastName: 'Test'
      }));
      localStorage.setItem('kmm_token', 'dummy');
      // Simulate that this user has already linked Spotify
      localStorage.setItem(`kmmSpotifyReady:${username}`, 'true');
      localStorage.setItem(`kmmSpotifyReadyTs:${username}`, Date.now().toString());
      // Clear visited flags
      localStorage.removeItem('kmmVisitedMatches');
      localStorage.removeItem('kmmVisitedEvents');
    }, { username });

    await page.goto(url + '/home');

    // Re-sync tile should show, and Link with Spotify tile should be absent
    await expect(page.getByRole('button', { name: 'Re-sync Spotify' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Link with Spotify' })).toHaveCount(0);
  });
});
