import { Component, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router, Routes } from '@angular/router';
import { SwUpdate, VersionReadyEvent } from '@angular/service-worker';
import { HttpClient } from '@angular/common/http';
import { AuthService } from './auth.service';
import { ThemeService } from './theme.service';
import { MatchService } from './match.service';
import { SpotifyConnectComponent } from './spotify-connect.component';
import { SpotifyCallbackComponent } from './spotify-callback.component';
import { PushService } from './services/push.service';
import { AppConfigService } from './services/app-config.service';
import { ToastService } from './services/toast.service';

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

  constructor(
    private auth: AuthService,
    private push: PushService,
    private router: Router,
    public theme: ThemeService,
    private matchService: MatchService,
    private swUpdate: SwUpdate,
    private http: HttpClient,
    private appCfg: AppConfigService,
    public toast: ToastService,
  ) {
    this.user = this.auth.user;
  }

  enablePush() { this.push.ensureSubscribed(); }

  testPush() {
    const { apiBaseUrl } = this.appCfg.get<any>();
    if (!apiBaseUrl) {
      this.toast.show('API base URL not configured.', { type: 'error', durationMs: 4000 });
      return;
    }
    this.http.post(`${apiBaseUrl}/push/test`, {}).subscribe({
      next: () => this.toast.show('Test notification sent', { type: 'success', durationMs: 3000 }),
      error: (err) => {
        console.error('Test push failed', err);
        const msg = (err?.status === 404) ? 'No subscription found for your account.' : 'Failed to send test notification.';
        this.toast.show(msg, { type: 'error', durationMs: 5000 });
      }
    });
  }

  logout(): void {
    this.auth.logout();
    this.matchService.clear();
    this.router.navigateByUrl('/');
  }

  toggleTheme(): void { this.theme.toggle(); }
  toggleMobileMenu(): void { this.mobileMenuOpen.set(!this.mobileMenuOpen()); }
  closeMobileMenu(): void { this.mobileMenuOpen.set(false); }

  ngOnInit(): void {
    this.matchService.fetchIfReady();

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