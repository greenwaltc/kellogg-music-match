import { Injectable } from '@angular/core';

export interface RuntimeConfig {
  artistMinCount?: number;
  artistMaxCount?: number;
  spotifyClientId?: string;
  spotifyRedirectUri?: string;
}

@Injectable({ providedIn: 'root' })
export class ConfigService {
  private cfgPromise: Promise<RuntimeConfig> | null = null;
  private current: RuntimeConfig | null = null;

  getConfig(): Promise<RuntimeConfig> {
    if (!this.cfgPromise) {
      this.cfgPromise = fetch('/config.json', { cache: 'no-cache' })
        .then(r => (r.ok ? r.json() : Promise.reject(new Error('config fetch failed'))))
  .then(json => json as RuntimeConfig)
  .catch(() => ({} as RuntimeConfig))
        .then(cfg => {
          this.current = cfg;
          return cfg;
        });
    }
    return this.cfgPromise;
  }

  get snapshot(): RuntimeConfig | null {
    return this.current;
  }
}