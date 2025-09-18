import { Injectable, signal } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { environment } from '../environments/environment';
import { AuthService } from './auth.service';

export interface MatchUser { 
  name: string; 
  overlap: number;
  score: number;
  artists: string[];
  [k: string]: any 
}

export interface Artist {
  id: number;
  name: string;
  created_at: string;
}

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
  const username = user.username;
  const headers = username ? new HttpHeaders({ 'X-User-Username': username }) : undefined;
  this.http.post<MatchUser[]>(this.url('/findMusicMatches'), { artists: user.artists }, { headers }).subscribe({
        next: (res: MatchUser[]) => { this.matches.set(res); this.loading.set(false); },
        error: () => { this.loading.set(false); }
      });
    }
  }

  private url(path: string): string { return `${this.apiBase}${path}`; }

  searchArtists(query: string, limit: number = 10): Promise<Artist[]> {
    const params = new URLSearchParams({ q: query, limit: limit.toString() });
    return this.http.get<Artist[]>(`${this.url('/artists/search')}?${params}`).toPromise().then((result: Artist[] | undefined) => result || []);
  }
}
