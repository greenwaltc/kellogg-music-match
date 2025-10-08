import { Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../environments/environment';
import { ApiBaseService } from './api-base.service';
import { AuthService } from './auth.service';

export interface MatchOverlap {
  name: string;
  anchorRank: number;
  otherRank: number;
}

export interface MatchUser { 
  name: string; 
  overlap: number;
  score: number;
  artists: string[]; // flat list still provided for backwards compatibility / quick display
  overlaps?: MatchOverlap[]; // new structured overlap data with ranks
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
  private overlapsLimit: number | null = null; // null means not sent

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

  setOverlapsLimit(limit: number | null) {
    if (limit !== null) {
      this.overlapsLimit = Math.max(1, limit);
      localStorage.setItem('overlapsLimit', this.overlapsLimit.toString());
    } else {
      this.overlapsLimit = null;
      localStorage.removeItem('overlapsLimit');
    }
  }

  private initStoredPrefs() {
    const stored = localStorage.getItem('overlapsLimit');
    if (stored) {
      const n = parseInt(stored, 10);
      if (!isNaN(n) && n > 0) this.overlapsLimit = n;
    }
  }

  fetchIfReady(): void {
    const currentUser = this.auth.user();
    
    // Clear matches if user has changed
    if (this.lastFetchedForUser && currentUser?.username !== this.lastFetchedForUser) {
      this.clear();
    }
    
    if (this.matches() || !currentUser) return;

    // Previously we required manual artists; backend now uses stored Spotify data and ignores body list.
    // So we always attempt a fetch for a logged-in user (sending any existing artists array or empty list).
    this.loading.set(true);
    const username = currentUser.username;
  const range = 'medium_term';
  const limit = 50; // increased default to show more potential matches
  if (this.overlapsLimit === null) this.initStoredPrefs();
  const params: Record<string,string> = { range, limit: limit.toString() };
  if (this.overlapsLimit) params['overlapsLimit'] = this.overlapsLimit.toString();
  const qp = new URLSearchParams(params).toString();
    const bodyArtists = Array.isArray(currentUser.artists) ? currentUser.artists : [];
    this.http.post<MatchUser[]>(this.url(`/findMusicMatches?${qp}`), { artists: bodyArtists }).subscribe({
      next: (res: MatchUser[]) => {
        this.matches.set(res);
        this.loading.set(false);
        this.lastFetchedForUser = username;
      },
      error: (err) => {
        console.warn('[MatchService] fetchIfReady failed', err);
        this.loading.set(false);
      }
    });
  }

  private url(path: string): string { return `${this.apiBase}${path}`; }

  // Explicit fetch with overrides for future UI controls
  fetch(range: 'short_term'|'medium_term'|'long_term' = 'medium_term', limit: number = 50, overlapsLimit?: number | null): void {
    const user = this.auth.user();
    if (!user) return; // allow empty artists list; backend relies on Spotify snapshot
    if (this.loading()) return; // avoid duplicate inflight
    this.loading.set(true);
    if (typeof overlapsLimit !== 'undefined') {
      this.setOverlapsLimit(overlapsLimit);
    } else if (this.overlapsLimit === null) {
      this.initStoredPrefs();
    }
    const params: Record<string,string> = { range, limit: Math.min(Math.max(1, limit), 50).toString() };
    if (this.overlapsLimit) params['overlapsLimit'] = this.overlapsLimit.toString();
    const qp = new URLSearchParams(params).toString();
    const bodyArtists = Array.isArray(user.artists) ? user.artists : [];
    this.http.post<MatchUser[]>(this.url(`/findMusicMatches?${qp}`), { artists: bodyArtists }).subscribe({
      next: (res) => { this.set(res); this.lastFetchedForUser = user.username; },
      error: () => { this.loading.set(false); }
    });
  }

  searchArtists(query: string, limit: number = 10): Promise<Artist[]> {
    const params = new URLSearchParams({ q: query, limit: limit.toString() });
    return this.http.get<Artist[]>(`${this.url('/artists/search')}?${params}`).toPromise().then((result: Artist[] | undefined) => result || []);
  }
}
