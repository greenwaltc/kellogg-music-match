import { Injectable } from '@angular/core';

export interface RuntimeConfig {
  apiBaseUrl: string;
  artistMinCount?: number;
  artistMaxCount?: number;
  spotifyClientId?: string;
  spotifyRedirectUri?: string;
}

@Injectable({ providedIn: 'root' })
export class ConfigService {
  private cfgPromise: Promise<RuntimeConfig> | null = null;

  getConfig(): Promise<RuntimeConfig> {
    if (!this.cfgPromise) {
      this.cfgPromise = fetch('/config.json', { cache: 'no-cache' })
        .then(r => r.json())
        .then(json => json as RuntimeConfig)
        .catch(() => ({ apiBaseUrl: '/api' }));
    }
    return this.cfgPromise;
  }
}