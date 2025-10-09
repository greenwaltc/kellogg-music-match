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
  // Flag indicating backend has Spotify data for this user (set after successful sync callback)
  spotifyReady = signal<boolean>(false);
  private readyTimestamp: number | null = null; // epoch ms when readiness established
  private freshnessWindowMs = 30_000; // within this window, force a cache-busting fetch on range change
  private probed = false; // whether we've attempted an auto-probe for existing data
  private rangeHistory: Record<string, MatchUser[]> = {}; // cache previous fetched matches per range
  private basisHistory: Record<string, Record<string, MatchUser[]>> = {}; // basis -> range -> matches
  basis = signal<'artists'|'tracks'>('artists');
  private basisReadyProbeAttempted = false;

  constructor(private http: HttpClient, private auth: AuthService, private api: ApiBaseService) {
    this.apiBase = this.api.baseUrl;
    // Initialize readiness from persisted flag (e.g., returning to app after prior sync)
    const storedReady = localStorage.getItem('kmmSpotifyReady');
    if (storedReady && storedReady === 'true') {
      this.spotifyReady.set(true);
      const ts = localStorage.getItem('kmmSpotifyReadyTs');
      if (ts) {
        const num = parseInt(ts, 10);
        if (!isNaN(num)) this.readyTimestamp = num;
      }
    }
    const storedBasis = localStorage.getItem('kmmMatchBasis');
    if (storedBasis === 'tracks' || storedBasis === 'artists') {
      this.basis.set(storedBasis as 'artists'|'tracks');
    }
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
    // If not marked ready yet (e.g., returning user before we persisted readiness), try a one-time probe
    if (!this.spotifyReady()) {
      if (!this.probed && !(window as any)['__karma__']) {
        this.probed = true;
        const username = currentUser.username;
      const params: Record<string,string> = { range: 'medium_term', limit: '1', probe: '1', basis: this.basis() };
        const qp = new URLSearchParams(params).toString();
        this.http.post<MatchUser[]>(this.url(`/findMusicMatches?${qp}`), { artists: [] }).subscribe({
          next: (res) => {
            // If request succeeded (even empty array) we assume user has authorized before; mark ready & refetch full.
            this.spotifyReady.set(true);
            this.readyTimestamp = Date.now();
            localStorage.setItem('kmmSpotifyReady', 'true');
            localStorage.setItem('kmmSpotifyReadyTs', this.readyTimestamp.toString());
            this.fetch('medium_term', 50, this.overlapsLimit, true);
            this.lastFetchedForUser = username;
          },
          error: () => { /* silently ignore; user likely truly not connected */ }
        });
      }
      return; // still exit; full fetch done after readiness
    }

    // Previously we required manual artists; backend now uses stored Spotify data and ignores body list.
    // So we always attempt a fetch for a logged-in user (sending any existing artists array or empty list).
    this.loading.set(true);
    const username = currentUser.username;
  const range = 'medium_term';
  const limit = 50; // increased default to show more potential matches
  if (this.overlapsLimit === null) this.initStoredPrefs();
  const params: Record<string,string> = { range, limit: limit.toString(), basis: this.basis() };
  if (this.overlapsLimit) params['overlapsLimit'] = this.overlapsLimit.toString();
  const qp = new URLSearchParams(params).toString();
    const bodyArtists = Array.isArray(currentUser.artists) ? currentUser.artists : [];
    this.http.post<MatchUser[]>(this.url(`/findMusicMatches?${qp}`), { artists: bodyArtists }).subscribe({
      next: (res: MatchUser[]) => {
  this.matches.set(res);
  if (!this.basisHistory[this.basis()]) this.basisHistory[this.basis()] = {};
  this.basisHistory[this.basis()][range] = res;
  // legacy single-level range history retained for backward logic
  this.rangeHistory[range] = res;
        this.loading.set(false);
        this.lastFetchedForUser = username;
      },
      error: (err) => {
        if (this.basis()==='tracks' && err?.error?.message === 'track-based matching disabled') {
          // auto-fallback to artists to stop spamming disabled endpoint
          this.basis.set('artists');
          localStorage.setItem('kmmMatchBasis','artists');
          this.loading.set(false);
          // retry once with artists
          queueMicrotask(()=> this.fetch('medium_term', 50, this.overlapsLimit, true));
          return;
        }
        console.warn('[MatchService] fetchIfReady failed', err);
        this.loading.set(false);
      }
    });
  }

  private url(path: string): string { return `${this.apiBase}${path}`; }

  // Explicit fetch with overrides for future UI controls
  fetch(range: 'short_term'|'medium_term'|'long_term' = 'medium_term', limit: number = 50, overlapsLimit?: number | null, forceFresh: boolean = false): void {
    const user = this.auth.user();
    if (!user) return; // allow empty artists list; backend relies on Spotify snapshot
    if (!this.spotifyReady()) return; // guard until Spotify sync complete
    if (this.loading()) return; // avoid duplicate inflight
    this.loading.set(true);
    if (typeof overlapsLimit !== 'undefined') {
      this.setOverlapsLimit(overlapsLimit);
    } else if (this.overlapsLimit === null) {
      this.initStoredPrefs();
    }
  const params: Record<string,string> = { range, limit: Math.min(Math.max(1, limit), 50).toString(), basis: this.basis() };
    // If explicitly forcing fresh OR within freshness window since readiness, add cache-busting param
    const now = Date.now();
    if (forceFresh || (this.readyTimestamp && (now - this.readyTimestamp) < this.freshnessWindowMs)) {
      params['fresh'] = now.toString();
    }
    if (this.overlapsLimit) params['overlapsLimit'] = this.overlapsLimit.toString();
    const qp = new URLSearchParams(params).toString();
    const bodyArtists = Array.isArray(user.artists) ? user.artists : [];
    // Check cache first
    const cached = this.basisHistory[this.basis()]?.[range];
    if (!forceFresh && cached) {
      this.matches.set(cached);
      this.loading.set(false);
      this.lastFetchedForUser = user.username;
      return;
    }
    this.http.post<MatchUser[]>(this.url(`/findMusicMatches?${qp}`), { artists: bodyArtists }).subscribe({
      next: (res) => { 
        this.set(res); 
        if (!this.basisHistory[this.basis()]) this.basisHistory[this.basis()] = {}; 
        this.basisHistory[this.basis()][range] = res;
        this.lastFetchedForUser = user.username; },
      error: (err) => { 
        if (this.basis()==='tracks' && err?.error?.message === 'track-based matching disabled') {
          this.basis.set('artists');
            localStorage.setItem('kmmMatchBasis','artists');
            this.loading.set(false);
            queueMicrotask(()=> this.fetch(range, limit, overlapsLimit, forceFresh));
            return;
        }
        this.loading.set(false); 
      }
    });
  }

  searchArtists(query: string, limit: number = 10): Promise<Artist[]> {
    const params = new URLSearchParams({ q: query, limit: limit.toString() });
    return this.http.get<Artist[]>(`${this.url('/artists/search')}?${params}`).toPromise().then((result: Artist[] | undefined) => result || []);
  }

  /** Mark that Spotify data is available and immediately refetch fresh matches. */
  markSpotifyReadyAndRefetch(range: 'short_term'|'medium_term'|'long_term' = 'medium_term') {
    this.spotifyReady.set(true);
    this.readyTimestamp = Date.now();
    localStorage.setItem('kmmSpotifyReady', 'true');
    localStorage.setItem('kmmSpotifyReadyTs', this.readyTimestamp.toString());
    // Clear any stale pre-spotify matches
    this.clear();
    // Trigger a forced fresh fetch (respect stored overlaps limit and default limit)
    this.fetch(range, 50, this.overlapsLimit, true);
  }

  /** Retrieve overlap count for a match from a previously cached range (if available). */
  previousOverlap(matchName: string, prevRange: 'short_term'|'medium_term'|'long_term'): number | null {
    const list = this.rangeHistory[prevRange];
    if (!list) return null;
    const m = list.find(mu => mu.name === matchName);
    return m ? m.overlap : null;
  }

  setBasis(b: 'artists'|'tracks') {
    if (this.basis() === b) return;
    this.basis.set(b);
    localStorage.setItem('kmmMatchBasis', b);
    // Attempt fetch for current default range if not cached
    const defaultRange: 'short_term'|'medium_term'|'long_term' = 'medium_term';
    if (!this.basisHistory[b]?.[defaultRange]) {
      this.fetch(defaultRange, 50, this.overlapsLimit, false);
    } else {
      // show cached immediately
      this.matches.set(this.basisHistory[b][defaultRange]);
    }
  }
}
