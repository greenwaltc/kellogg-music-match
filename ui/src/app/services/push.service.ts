import { Injectable, inject } from '@angular/core';
import { SwPush } from '@angular/service-worker';
import { HttpClient } from '@angular/common/http';
import { AppConfigService, AppConfig } from './app-config.service';

type ExtendedConfig = AppConfig;

@Injectable({ providedIn: 'root' })
export class PushService {
  private swPush = inject(SwPush);
  private http = inject(HttpClient);
  private cfg = inject(AppConfigService);

  constructor() {
    // Log push messages delivered to the SW and forwarded to the app
    this.swPush.messages.subscribe(msg => {
      console.log('[Push] message', msg);
    });
    // Log notification clicks
    this.swPush.notificationClicks.subscribe(event => {
      console.log('[Push] notification click', event);
    });
  }

  private normalizeVapidKey(k: string | undefined | null): string {
    // Trim, strip whitespace, convert to URL-safe base64 variant, remove padding
    const raw = (k ?? '').trim().replace(/\s+/g, '');
    if (!raw) return '';
    if (raw.startsWith('-----BEGIN')) {
      // PEM provided instead of base64url public key
      return raw;
    }
    return raw.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
  }

  private base64UrlToUint8Array(base64Url: string): Uint8Array {
    const pad = '='.repeat((4 - (base64Url.length % 4)) % 4);
    const base64 = (base64Url + pad).replace(/-/g, '+').replace(/_/g, '/');
    const raw = atob(base64);
    const out = new Uint8Array(raw.length);
    for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
    return out;
  }

  private isValidVapidPublicKey(base64UrlKey: string): boolean {
    try {
      const u8 = this.base64UrlToUint8Array(base64UrlKey);
      // Expected uncompressed EC point for P-256 (65 bytes, first byte 0x04)
      return u8.length === 65 && u8[0] === 0x04;
    } catch {
      return false;
    }
  }

  async ensureSubscribed(): Promise<PushSubscription | null> {
    console.log('[Push] ensureSubscribed() clicked');

    if (!('Notification' in window)) {
      console.warn('[Push] Notifications unsupported in this browser.');
      return null;
    }

    // 1) Ask for permission first (must be triggered by a user gesture)
    let perm = Notification.permission;
    console.log('[Push] current permission:', perm);
    if (perm === 'default') {
      perm = await Notification.requestPermission();
      console.log('[Push] permission result:', perm);
    }
    if (perm !== 'granted') {
      console.warn('[Push] Permission not granted:', perm);
      return null;
    }

    // 2) Ensure the SW is ready (covers delayed registration strategies)
    try {
      await navigator.serviceWorker.ready;
    } catch {}

    // 3) Now try to subscribe with Angular SwPush
    if (!this.swPush.isEnabled) {
      console.warn('[Push] SwPush not enabled yet; try again after SW is active.');
      return null;
    }

    const { vapidPublicKey, apiBaseUrl } = this.cfg.get<ExtendedConfig>();
    const serverPublicKey = this.normalizeVapidKey(vapidPublicKey);
    if (!serverPublicKey) {
      console.error('[Push] Missing vapidPublicKey in config.json');
      return null;
    }
    if (serverPublicKey.startsWith('-----BEGIN')) {
      console.error('[Push] VAPID key appears to be a PEM block. Provide the base64url-encoded public key string (e.g., from web-push generate-vapid-keys).');
      return null;
    }
    if (!this.isValidVapidPublicKey(serverPublicKey)) {
      console.error('[Push] VAPID public key is not a valid base64url-encoded P-256 public key. Ensure you are using the PUBLIC key from `web-push generate-vapid-keys` (the long string starting with "B...") and not the PEM.');
      return null;
    }

    try {
      // If there's an existing subscription (possibly with a different key), unsubscribe first.
      try {
        const reg = await navigator.serviceWorker.ready;
        const existing = await reg.pushManager.getSubscription();
        if (existing) {
          const ok = await existing.unsubscribe();
          console.log('[Push] Unsubscribed existing subscription:', ok);
        }
      } catch {}

      const sub = await this.swPush.requestSubscription({ serverPublicKey });
      console.log('[Push] Subscription created:', JSON.stringify(sub));

      if (apiBaseUrl) {
        try {
          await this.http.post(`${apiBaseUrl}/push/subscribe`, sub).toPromise();
          console.log('[Push] Subscription sent to server.');
        } catch {
          console.warn('[Push] Could not send subscription to server yet.');
        }
      }
      return sub;
    } catch (err) {
      console.error('[Push] Subscription failed:', err);
      return null;
    }
  }
}