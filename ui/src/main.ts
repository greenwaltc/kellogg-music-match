import { bootstrapApplication } from '@angular/platform-browser';
import { provideAnimations } from '@angular/platform-browser/animations';
import { provideHttpClient, withFetch } from '@angular/common/http';
import { provideRouter, Routes } from '@angular/router';
import { AppComponent } from './app/app.component';
import { LoginComponent } from './app/login.component';
import { ArtistsComponent } from './app/artists.component';
import { MatchesComponent } from './app/matches.component';
import { FeedbackComponent } from './app/feedback.component';
import { RoadmapComponent } from './app/roadmap.component';
import { authGuard } from './app/auth.guard';

declare global {
  interface Window {
    __kmmConfig?: { apiBaseUrl?: string };
  }
}

const routes: Routes = [
  { path: '', component: LoginComponent },
  { path: 'artists', component: ArtistsComponent, canActivate: [authGuard] },
  { path: 'matches', component: MatchesComponent, canActivate: [authGuard] },
  { path: 'feedback', component: FeedbackComponent, canActivate: [authGuard] },
  { path: 'roadmap', component: RoadmapComponent, canActivate: [authGuard] },
  { path: '**', redirectTo: '' }
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
      provideHttpClient(withFetch()),
      provideRouter(routes)
    ]
  }).catch((err: unknown) => console.error(err));
}

loadConfigAndBootstrap();
