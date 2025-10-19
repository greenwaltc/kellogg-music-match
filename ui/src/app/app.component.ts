import { Component, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router, Routes } from '@angular/router';
import { SwUpdate, VersionReadyEvent } from '@angular/service-worker';
import { HttpClient } from '@angular/common/http';
import { ApiBaseService } from './api-base.service';
import { AuthService } from './auth.service';
import { ThemeService } from './theme.service';
import { MatchService } from './match.service';
import { SpotifyConnectComponent } from './spotify-connect.component';
import { SpotifyCallbackComponent } from './spotify-callback.component';
import { PushService } from './services/push.service';
import { ToastService } from './services/toast.service';
import { NavStateService } from './nav-state.service';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterModule],
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent implements OnInit {
  title = 'Kellogg Music Match';
  user = signal<any>(null);
  isLoggedIn = computed(() => !!this.user());
  mobileMenuOpen = signal(false);
  notifPermission = signal<NotificationPermission>(typeof Notification !== 'undefined' ? Notification.permission : 'default');
  // Nav visibility state persisted across sessions
  constructor(
    private auth: AuthService,
    private push: PushService,
    private router: Router,
    public theme: ThemeService,
    private matchService: MatchService,
    private swUpdate: SwUpdate,
    private http: HttpClient,
    public toast: ToastService,
    private api: ApiBaseService,
    public nav: NavStateService,
  ) {
    this.user = this.auth.user;
    // Expose nav reset for post-auth side-effects without tight coupling
    try { (window as any).ngNavState = this.nav; } catch {}
  }
  // visited flags now come from NavStateService; template references nav.visitedMatches()/visitedEvents()


  testPush() {
    this.http.post(this.api.url('/push/test'), {}).subscribe({
      next: () => this.toast.show('Test notification sent', { type: 'success', durationMs: 3000 }),
      error: (err) => {
        console.error('Test push failed', err);
        const msg = (err?.status === 404) ? 'No subscription found for your account.' : 'Failed to send test notification.';
        this.toast.show(msg, { type: 'error', durationMs: 5000 });
      }
    });
  }

  testPushEnqueue() {
    this.http.post(this.api.url('/push/test/enqueue'), {}).subscribe({
      next: () => this.toast.show('Async test notification enqueued', { type: 'success', durationMs: 3000 }),
      error: (err) => {
        console.error('Enqueue test push failed', err);
        let msg = 'Failed to enqueue notification.';
        if (err?.status === 429) msg = 'Too many requests. Try again in a minute.';
        if (err?.status === 412) msg = 'Push not configured on the server.';
        this.toast.show(msg, { type: 'error', durationMs: 5000 });
      }
    });
  }

  // Debug visibility guard: enable by running in console `localStorage.setItem('kmm_debug_ui','1')`
  debugVisible(): boolean {
    try { return !!localStorage.getItem('kmm_debug_ui'); } catch { return false; }
  }

  // Call backend VAPID debug endpoint and show summary
  vapidDebug() {
    // Try to include bearer token if present; otherwise rely on cookies
    let headers: any = {};
    try {
      const tok = localStorage.getItem('kmm_token');
      if (tok) headers['Authorization'] = `Bearer ${tok}`;
    } catch {}
    this.http.get<any>(this.api.url('/push/debug/vapid'), { headers, withCredentials: true }).subscribe({
      next: (res) => {
        console.log('[VAPID Debug]', res);
        const match = res?.pairMatch ? 'match' : 'MISMATCH';
        const msg = `VAPID ${match}. pub:${res?.publicKeyPrefix || ''} derived:${res?.derivedPublicPrefix || ''} host:${res?.endpointHost || ''} aud:${res?.audienceComputed || ''}${res?.usedFallbackAny ? ' (fallback)' : ''}`;
        this.toast.show(msg, { type: res?.pairMatch ? 'success' : 'warning', durationMs: 6000 });
      },
      error: (err) => {
        console.error('VAPID debug failed', err);
        let msg = 'VAPID debug failed';
        if (err?.status === 401) msg = 'Unauthorized (login required)';
        if (err?.status === 412) msg = 'Push disabled on server';
        this.toast.show(msg, { type: 'error', durationMs: 5000 });
      }
    });
  }

  logout(): void {
    this.auth.logout();
    this.nav.resetVisited();
    this.matchService.clear();
    this.router.navigateByUrl('/');
  }

  toggleTheme(): void { this.theme.toggle(); }
  toggleMobileMenu(): void { this.mobileMenuOpen.set(!this.mobileMenuOpen()); }
  closeMobileMenu(): void { this.mobileMenuOpen.set(false); }

  ngOnInit(): void {
    this.matchService.fetchIfReady();

    // Track router navigation to set visited flags
    this.router.events.subscribe(() => {
      const url = this.router.url || '';
      if (url.startsWith('/matches') || url.startsWith('/chicago-events')) {
        this.nav.markAllPrimaryVisited();
      }
    });

    // Track notification permission changes (best-effort)
    try {
      if (typeof window !== 'undefined') {
        const updatePerm = () => this.notifPermission.set(Notification.permission);
        document.addEventListener('visibilitychange', updatePerm);
        // Also update once on boot after a tick
        setTimeout(updatePerm, 0);
      }
    } catch {}

    if (this.swUpdate.isEnabled) {
      this.swUpdate.versionUpdates.subscribe(evt => {
        const e = evt as VersionReadyEvent;
        if (e.type === 'VERSION_READY') {
          this.toast.show('A new version is available.', {
            type: 'info',
            actionLabel: 'Reload now',
            action: () => this.swUpdate.activateUpdate().then(() => document.location.reload()),
            durationMs: 0,
          });
        }
      });
    }
  }
}

// Provide route definitions (Angular standalone style) - This snippet is illustrative; actual route wiring likely elsewhere.
export const SPOTIFY_ROUTES: Routes = [
  { path: 'spotify/connect', component: SpotifyConnectComponent },
  { path: 'spotify/callback', component: SpotifyCallbackComponent }
];