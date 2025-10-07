import { bootstrapApplication } from '@angular/platform-browser';
import { provideAnimations } from '@angular/platform-browser/animations';
import { provideHttpClient, withFetch, withInterceptors } from '@angular/common/http';
import { provideRouter, Routes } from '@angular/router';
import { AppComponent } from './app/app.component';
import { LoginComponent } from './app/login.component';
import { ArtistsComponent } from './app/artists.component';
import { MatchesComponent } from './app/matches.component';
import { FeedbackComponent } from './app/feedback.component';
import { RoadmapComponent } from './app/roadmap.component';
import { ChicagoEventsComponent } from './app/chicago-events.component';
import { SpotifyConnectComponent } from './app/spotify-connect.component';
import { SpotifyCallbackComponent } from './app/spotify-callback.component';
import { authGuard } from './app/auth.guard';
import { jwtInterceptor } from './app/jwt.interceptor';
import { loginRedirectGuard } from './app/login-redirect.guard';

declare global {
  interface Window {
    __kmmConfig?: { apiBaseUrl?: string };
  }
}

const routes: Routes = [
  { path: '', component: LoginComponent, canActivate: [loginRedirectGuard] },
  { path: 'login', redirectTo: '', pathMatch: 'full' },
  { path: 'matches', component: MatchesComponent, canActivate: [authGuard] },
  { path: 'feedback', component: FeedbackComponent, canActivate: [authGuard] },
  { path: 'roadmap', component: RoadmapComponent, canActivate: [authGuard] },
  { path: 'chicago-events', component: ChicagoEventsComponent, canActivate: [authGuard] },
  { path: 'spotify/callback', component: SpotifyCallbackComponent },
  { path: '**', redirectTo: '' }
  // Artists & standalone Spotify connect page removed (Spotify connect now on Matches page)
];

// Wait for config to load before bootstrapping Angular
async function loadConfigAndBootstrap() {
  try {
    // Give the config loading script some time to complete
    let retries = 10;
    while (retries > 0 && (!window.__kmmConfig?.apiBaseUrl || window.__kmmConfig.apiBaseUrl === '')) {
      await new Promise(resolve => setTimeout(resolve, 100));
      retries--;
    }
  } catch (err) {
    console.warn('Config loading failed:', err);
  }

  bootstrapApplication(AppComponent, {
    providers: [
      provideAnimations(),
      provideHttpClient(withFetch(), withInterceptors([jwtInterceptor])),
      provideRouter(routes)
    ]
  }).catch((err: unknown) => console.error(err));
}

loadConfigAndBootstrap();
