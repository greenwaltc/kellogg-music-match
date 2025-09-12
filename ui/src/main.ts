import { bootstrapApplication } from '@angular/platform-browser';
import { provideAnimations } from '@angular/platform-browser/animations';
import { provideHttpClient, withFetch } from '@angular/common/http';
import { provideRouter, Routes } from '@angular/router';
import { AppComponent } from './app/app.component';
import { LoginComponent } from './app/login.component';
import { ArtistsComponent } from './app/artists.component';
import { MatchesComponent } from './app/matches.component';
import { authGuard } from './app/auth.guard';

const routes: Routes = [
  { path: '', component: LoginComponent },
  { path: 'artists', component: ArtistsComponent, canActivate: [authGuard] },
  { path: 'matches', component: MatchesComponent, canActivate: [authGuard] },
  { path: '**', redirectTo: '' }
];

bootstrapApplication(AppComponent, {
  providers: [
    provideAnimations(),
    provideHttpClient(withFetch()),
    provideRouter(routes)
  ]
}).catch((err: unknown) => console.error(err));
