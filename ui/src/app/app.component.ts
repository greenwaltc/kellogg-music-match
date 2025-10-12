import { Component, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router, Routes } from '@angular/router';
import { SwUpdate, VersionReadyEvent } from '@angular/service-worker';
import { AuthService } from './auth.service';
import { ThemeService } from './theme.service';
import { MatchService } from './match.service';
import { SpotifyConnectComponent } from './spotify-connect.component';
import { SpotifyCallbackComponent } from './spotify-callback.component';
import { PushService } from './services/push.service';

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
    private swUpdate: SwUpdate
  ) {
    this.user = this.auth.user;
  }

  enablePush() { this.push.ensureSubscribed(); }

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
        if ((evt as VersionReadyEvent).type === 'VERSION_READY') {
          const accept = confirm('A new version of the app is available. Reload now?');
          if (accept) {
            this.swUpdate.activateUpdate().then(() => document.location.reload());
          }
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