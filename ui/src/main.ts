import { provideAnimations } from '@angular/platform-browser/animations';
import { provideHttpClient, withFetch, withInterceptors } from '@angular/common/http';
import { provideRouter, Routes } from '@angular/router';
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
import { provideServiceWorker } from '@angular/service-worker';
import { APP_INITIALIZER } from '@angular/core';
import { bootstrapApplication } from '@angular/platform-browser';
import { AppComponent } from './app/app.component';
import { AppConfigService } from './app/services/app-config.service';
import { isDevMode } from '@angular/core';
import { ConfigService } from './app/config.service';

// Ensure your existing providers remain; we add APP_INITIALIZER
function initConfig(cfg: AppConfigService) {
  return () => cfg.load();
}

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

bootstrapApplication(AppComponent, {
  providers: [
    provideAnimations(),
    provideHttpClient(withFetch(), withInterceptors([jwtInterceptor])),
    provideRouter(routes),
    provideServiceWorker('ngsw-worker.js', {
      enabled: !isDevMode(),
      registrationStrategy: 'registerWhenStable:30000'
    }),
    { provide: APP_INITIALIZER, useFactory: initConfig, deps: [AppConfigService], multi: true }
  ]
}).catch((err: unknown) => console.error(err));
