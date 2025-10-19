import { Injectable, inject, signal } from '@angular/core';
import { SwPush } from '@angular/service-worker';
import { HttpClient } from '@angular/common/http';
import { AppConfigService, AppConfig } from './app-config.service';
import { ApiBaseService } from '../api-base.service';
import { ToastService } from './toast.service';

type ExtendedConfig = AppConfig;

@Injectable({ providedIn: 'root' })
export class PushService {
  private swPush = inject(SwPush);
  private http = inject(HttpClient);
  private cfg = inject(AppConfigService);
  private toast = inject(ToastService);
  private api = inject(ApiBaseService);
  private lastSubscription: PushSubscription | null = null;
  private retryTimer: any = null;
  private retryAttempts = 0;
  // Whether this device has an active push subscription (and permission granted)
  hasDeviceSubscription = signal<boolean>(false);

  constructor() {
    // Log push messages delivered to the SW and forwarded to the app
    this.swPush.messages.subscribe((msg: any) => {
      console.log('[Push] message', msg);
      // Fallback: only show a page-level notification if the SW did NOT include a notification.
      // This avoids duplicate notifications when the SW already displayed one.
      try {
        const noti = msg?.notification;
        if (!noti && typeof Notification !== 'undefined' && Notification.permission === 'granted') {
          const d = msg?.data || {};
          const n = new Notification(d.title ?? 'Notification', {
            body: d.body,
            icon: d.icon,
            badge: d.badge,
            data: d,
          });
          n.onclick = () => {
            try {
              const url = (d && (d.url || d.link)) || '/';
              if (url && typeof window !== 'undefined') {
                window.focus?.();
                window.location.assign(url);
              }
            } catch {}
          };
        }
      } catch {}
    });
    // Log notification clicks and navigate when possible
    this.swPush.notificationClicks.subscribe((event: any) => {
      console.log('[Push] notification click', event);
      try {
        const url = event?.notification?.data?.url || event?.notification?.data?.link;
        if (url && typeof window !== 'undefined') {
          window.focus?.();
          window.location.assign(url);
        }
      } catch {}
    });
  }

  /** Check if this device has an active subscription and update the signal. */
  async refreshDeviceSubscriptionFlag(): Promise<void> {
    try {
      if (!('Notification' in window)) { this.hasDeviceSubscription.set(false); return; }
      if (Notification.permission !== 'granted') { this.hasDeviceSubscription.set(false); return; }
      const reg = await navigator.serviceWorker.ready;
      const existing = await reg.pushManager.getSubscription();
      this.hasDeviceSubscription.set(!!existing);
    } catch {
      this.hasDeviceSubscription.set(false);
    }
  }

  // Fetch a fresh VAPID public key bypassing SW cache; fallback to current in-memory config
  private async getFreshVapidKey(): Promise<string> {
    try {
      const res = await fetch('/config.json?ngsw-bypass=true', { cache: 'no-store' });
      if (res.ok) {
        const fresh = await res.json();
        const key = this.normalizeVapidKey(fresh?.vapidPublicKey);
        if (key) return key;
      }
    } catch {}
    const { vapidPublicKey } = this.cfg.get<ExtendedConfig>();
    return this.normalizeVapidKey(vapidPublicKey);
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
      // Immediately reflect permission change in UI
      if (perm === 'granted') {
        // We may not have a subscription yet, but permission is granted.
        // Trigger a refresh to update device subscription flag ASAP.
        try { await this.refreshDeviceSubscriptionFlag(); } catch {}
      }
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

    // Always prefer a fresh key from the network to avoid stale SW-cached config
    const serverPublicKey = await this.getFreshVapidKey();
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

    // Log fingerprint to help correlate which key is used client-side (first 8 chars)
    try { console.log('[Push] Using VAPID key (prefix):', serverPublicKey.substring(0, 12)); } catch {}

    const sub = await this.swPush.requestSubscription({ serverPublicKey });
      console.log('[Push] Subscription created:', JSON.stringify(sub));

      try {
        await this.http.post(this.api.url('/push/subscribe'), sub).toPromise();
        console.log('[Push] Subscription sent to server.');
        this.toast.show('Notifications enabled', { type: 'success', durationMs: 3000 });
        this.hasDeviceSubscription.set(true);
      } catch {
        console.warn('[Push] Could not send subscription to server yet.');
        this.toast.show('Subscribed locally, but server not reachable. Will retry later.', { type: 'warning', durationMs: 5000 });
        this.lastSubscription = sub;
        this.scheduleRetry();
        this.hasDeviceSubscription.set(true);
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
    try {
      await this.http.post(this.api.url('/push/subscribe'), sub).toPromise();
      this.retryAttempts = 0;
      this.lastSubscription = null;
      this.toast.show('Notifications enabled (server connected)', { type: 'success', durationMs: 2500 });
      this.hasDeviceSubscription.set(true);
    } catch {
      this.retryAttempts++;
      this.scheduleRetry();
    }
  }
}