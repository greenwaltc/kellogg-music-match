import { Injectable } from '@angular/core';

export interface AppConfig {
  apiBaseUrl: string;
  featureFlags?: Record<string, any>;
}

const CONFIG_URL = '/config.json';
const BYPASS_URL = `${CONFIG_URL}?ngsw-bypass=true`;

@Injectable({ providedIn: 'root' })
export class AppConfigService {
  private config: AppConfig = { apiBaseUrl: '', featureFlags: {} };

  async load(): Promise<void> {
    // Try SW cache first (works offline if previously cached)
    try {
      if ('caches' in window) {
        const cached = await caches.match(CONFIG_URL, { ignoreSearch: true });
        if (cached) {
          this.config = await cached.json();
          // Refresh later, only when online, and bypass SW
          this.whenOnline(() => this.refreshInBackground());
          return;
        }
      }
    } catch {
      // ignore cache errors
    }

    // No cache on cold start: use safe defaults and hydrate later only when online
    this.config = { apiBaseUrl: '', featureFlags: {} };
    this.whenOnline(() => this.hydrateFromNetwork());
  }

  get<T extends AppConfig = AppConfig>(): T {
    return this.config as T;
  }

  private async hydrateFromNetwork() {
    try {
      const res = await fetch(BYPASS_URL, { cache: 'no-store' });
      if (!res.ok) return;
      this.config = await res.json();
    } catch {}
  }

  private async refreshInBackground() {
    try {
      await fetch(BYPASS_URL, { cache: 'no-store' });
    } catch {}
  }

  private whenOnline(fn: () => void) {
    if (navigator.onLine) {
      // Run shortly after bootstrap to avoid blocking
      setTimeout(fn, 0);
      return;
    }
    const h = () => { window.removeEventListener('online', h); fn(); };
    window.addEventListener('online', h, { once: true });
  }
}