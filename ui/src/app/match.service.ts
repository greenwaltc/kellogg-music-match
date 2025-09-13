import { Injectable, signal } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { environment } from '../environments/environment';
import { AuthService } from './auth.service';

export interface MatchUser { name: string; [k: string]: any }

@Injectable({ providedIn: 'root' })
export class MatchService {
  matches = signal<MatchUser[] | null>(null);
  loading = signal(false);
  private apiBase = window.__kmmConfig?.apiBaseUrl || environment.apiBaseUrl;

  constructor(private http: HttpClient, private auth: AuthService) {}

  set(matches: MatchUser[]): void { this.matches.set(matches); }
  clear(): void { this.matches.set(null); }

  fetchIfReady(): void {
    if (this.matches() || !this.auth.user()) return;
    const user = this.auth.user();
    if (user?.artists?.length) {
      this.loading.set(true);
  const email = user.email;
  const headers = email ? new HttpHeaders({ 'X-User-Email': email }) : undefined;
  this.http.post<MatchUser[]>(this.url('/findMusicMatches'), { artists: user.artists }, { headers }).subscribe({
        next: (res) => { this.matches.set(res); this.loading.set(false); },
        error: () => { this.loading.set(false); }
      });
    }
  }

  private url(path: string): string { return `${this.apiBase}${path}`; }
}
