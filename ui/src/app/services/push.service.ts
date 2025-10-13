import { Injectable, inject } from '@angular/core';
import { SwPush } from '@angular/service-worker';
import { HttpClient } from '@angular/common/http';
import { AppConfigService, AppConfig } from './app-config.service';
import { ToastService } from './toast.service';

type ExtendedConfig = AppConfig;

@Injectable({ providedIn: 'root' })
export class PushService {
  private swPush = inject(SwPush);
  private http = inject(HttpClient);
  private cfg = inject(AppConfigService);
  private toast = inject(ToastService);
  private lastSubscription: PushSubscription | null = null;
  private retryTimer: any = null;
  private retryAttempts = 0;

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
      this.toast.show('Notifications are not supported in this browser.', { type: 'warning', durationMs: 4000 });
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
      // Offer a retry if the user changed permissions in settings
      this.toast.show('Notifications are disabled. Enable in your browser settings and retry.', {
        type: 'warning',
        actionLabel: 'Retry',
        action: () => this.ensureSubscribed(),
        durationMs: 0,
      });
      return null;
    }

    // 2) Ensure the SW is ready (covers delayed registration strategies)
    try {
      await navigator.serviceWorker.ready;
    } catch {}

    // 3) Now try to subscribe with Angular SwPush
    if (!this.swPush.isEnabled) {
      console.warn('[Push] SwPush not enabled yet; try again after SW is active.');
      this.toast.show('Service worker not ready yet. Please try again in a moment.', { type: 'info', durationMs: 3000 });
      return null;
    }

    const { vapidPublicKey, apiBaseUrl } = this.cfg.get<ExtendedConfig>();
    const serverPublicKey = this.normalizeVapidKey(vapidPublicKey);
    if (!serverPublicKey) {
      console.error('[Push] Missing vapidPublicKey in config.json');
      this.toast.show('Push is not configured on the server (missing VAPID key).', { type: 'error', durationMs: 5000 });
      return null;
    }
    if (serverPublicKey.startsWith('-----BEGIN')) {
      console.error('[Push] VAPID key appears to be a PEM block. Provide the base64url-encoded public key string (e.g., from web-push generate-vapid-keys).');
      this.toast.show('Invalid VAPID key format on server.', { type: 'error', durationMs: 5000 });
      return null;
    }
    if (!this.isValidVapidPublicKey(serverPublicKey)) {
      console.error('[Push] VAPID public key is not a valid base64url-encoded P-256 public key. Ensure you are using the PUBLIC key from `web-push generate-vapid-keys` (the long string starting with "B...") and not the PEM.');
      this.toast.show('Server VAPID key is invalid.', { type: 'error', durationMs: 5000 });
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
          this.toast.show('Notifications enabled', { type: 'success', durationMs: 3000 });
        } catch {
          console.warn('[Push] Could not send subscription to server yet.');
          this.toast.show('Subscribed locally, but server not reachable. Will retry later.', { type: 'warning', durationMs: 5000 });
          this.lastSubscription = sub;
          this.scheduleRetry();
        }
      }
      return sub;
    } catch (err) {
      console.error('[Push] Subscription failed:', err);
      this.toast.show('Failed to enable notifications. Please try again.', { type: 'error', durationMs: 5000, actionLabel: 'Retry', action: () => this.ensureSubscribed() });
      return null;
    }
  }

  private scheduleRetry() {
    if (this.retryTimer) return;
    const base = 5000; // 5s base
    const max = 60000; // cap at 60s
    const delay = Math.min(base * Math.pow(2, this.retryAttempts), max);
    this.retryTimer = setTimeout(() => {
      this.retryTimer = null;
      this.retryPost();
    }, delay);
  }

  private async retryPost() {
    const sub = this.lastSubscription;
    if (!sub) return;
    const { apiBaseUrl } = this.cfg.get<ExtendedConfig>();
    if (!apiBaseUrl) return;
    try {
      await this.http.post(`${apiBaseUrl}/push/subscribe`, sub).toPromise();
      this.retryAttempts = 0;
      this.lastSubscription = null;
      this.toast.show('Notifications enabled (server connected)', { type: 'success', durationMs: 2500 });
    } catch {
      this.retryAttempts++;
      this.scheduleRetry();
    }
  }
}