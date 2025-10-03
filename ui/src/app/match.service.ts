import { Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../environments/environment';
import { ApiBaseService } from './api-base.service';
import { AuthService } from './auth.service';

export interface MatchUser { 
  name: string; 
  overlap: number;
  score: number;
  artists: string[];
  program?: string;
  graduationYear?: number;
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
  private apiBase: string;
  private lastFetchedForUser: string | null = null;

  constructor(private http: HttpClient, private auth: AuthService, private api: ApiBaseService) {
    this.apiBase = this.api.baseUrl;
  }

  set(matches: MatchUser[]): void { 
    this.matches.set(matches); 
    this.loading.set(false);
  }
  clear(): void { 
    this.matches.set(null); 
    this.lastFetchedForUser = null;
  }

  fetchIfReady(): void {
    const currentUser = this.auth.user();
    
    // Clear matches if user has changed
    if (this.lastFetchedForUser && currentUser?.username !== this.lastFetchedForUser) {
      this.clear();
    }
    
    if (this.matches() || !currentUser) return;
    
    if (currentUser?.artists?.length) {
      this.loading.set(true);
      const username = currentUser.username;
      // JWT interceptor will automatically add the Authorization header
      this.http.post<MatchUser[]>(this.url('/findMusicMatches'), { artists: currentUser.artists }).subscribe({
        next: (res: MatchUser[]) => { 
          this.matches.set(res); 
          this.loading.set(false);
          this.lastFetchedForUser = username;
        },
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
