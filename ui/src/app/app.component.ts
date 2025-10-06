import { Component, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router, Routes } from '@angular/router';
import { AuthService } from './auth.service';
import { ThemeService } from './theme.service';
import { MatchService } from './match.service';
import { SpotifyConnectComponent } from './spotify-connect.component';
import { SpotifyCallbackComponent } from './spotify-callback.component';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterModule],
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent {
  title = 'Kellogg Music Match';
  user = signal<any>(null);
  isLoggedIn = computed(() => !!this.user());
  mobileMenuOpen = signal(false);

  constructor(private auth: AuthService, private router: Router, public theme: ThemeService, private matchService: MatchService) {
    this.user = this.auth.user; // assign after DI
  }

  logout(): void {
    this.auth.logout();
    this.matchService.clear();
    this.router.navigateByUrl('/');
  }

  toggleTheme(): void { this.theme.toggle(); }

  toggleMobileMenu(): void {
    this.mobileMenuOpen.set(!this.mobileMenuOpen());
  }

  closeMobileMenu(): void {
    this.mobileMenuOpen.set(false);
  }

  ngOnInit(): void { this.matchService.fetchIfReady(); }
}

// Provide route definitions (Angular standalone style) - This snippet is illustrative; actual route wiring likely elsewhere.
export const SPOTIFY_ROUTES: Routes = [
  { path: 'spotify/connect', component: SpotifyConnectComponent },
  { path: 'spotify/callback', component: SpotifyCallbackComponent }
];
