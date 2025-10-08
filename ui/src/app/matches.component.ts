import { Component, effect, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { MatchService } from './match.service';
import { SpotifyService } from './spotify.service';
import { SimilarityMeterComponent } from './similarity-meter.component';
import { TooltipDirective } from './tooltip.directive';

@Component({
  selector: 'app-matches',
  standalone: true,
  imports: [CommonModule, FormsModule, SimilarityMeterComponent, TooltipDirective],
  template: `
  <section class="matches-page">
    <h2>Your Top Music Matches</h2>
    <div class="controls-bar">
      <label>
        Results Limit:
        <select [(ngModel)]="limit" (change)="onLimitChange()">
          <option *ngFor="let opt of limitOptions" [value]="opt">{{ opt }}</option>
        </select>
      </label>
      <label>
        Overlaps Limit:
        <select [(ngModel)]="overlapsLimit" (change)="onOverlapsLimitChange()" appTooltip="Max overlap entries per match (server may cap)">
          <option [ngValue]="null">All</option>
          <option *ngFor="let opt of overlapsLimitOptions" [value]="opt">{{ opt }}</option>
        </select>
      </label>
      <label>
        Debounce (ms):
        <input type="number" min="0" max="3000" [(ngModel)]="debounceMs" (change)="onDebounceChange()" class="debounce-input" />
      </label>
      <button type="button" class="reset-btn" (click)="resetControls()" appTooltip="Reset limit & debounce to defaults (50 / 400ms)">Reset</button>
      <div class="range-toggle" role="group" aria-label="Time Range">
        <button type="button" 
                *ngFor="let r of ranges" 
                (click)="selectRange(r.value)" 
                [class.active]="range===r.value" 
                [attr.aria-pressed]="range===r.value"
                class="range-btn" 
                appTooltip="{{ r.tooltip }}">
          {{ r.label }}
        </button>
      </div>
      <div class="range-subtitle">Listening window: <strong>{{ rangeLabel() }}</strong>
        <span class="info-icon small" appTooltip="Spotify provides 3 windows for your top artists:\nShort term ≈ last 4 weeks\nMedium term ≈ last 6 months\nLong term ≈ several years (weighted toward the past year)\nSelecting a different window recalculates overlap & rank positions." aria-label="Spotify time range info">i</span>
      </div>
    </div>
    <div class="spotify-connect-bar">
      <button type="button" (click)="connectSpotify()" class="spotify-btn">🎧 Connect / Sync Spotify</button>
    </div>
    <p class="note">Tip: check back here often! As more Kellogg students register, we build better matches.</p>
    
    <!-- Loading indicators -->
    <div *ngIf="matches.loading() && !matches.matches()" class="loading-container">
      <div class="loading-spinner"></div>
      <p class="loading-text">Finding your music matches...</p>
    </div>
    <div *ngIf="matches.loading() && matches.matches()" class="skeleton-list" aria-label="Loading updated matches">
      <div class="skeleton-card" *ngFor="let s of skeletonArray">
        <div class="skeleton-header">
          <div class="sk-block sk-title shimmer"></div>
          <div class="sk-meter shimmer"></div>
        </div>
        <div class="skeleton-body">
          <div class="sk-tags">
            <span class="sk-tag shimmer" *ngFor="let t of tagSkeleton"></span>
          </div>
            <div class="sk-stats">
              <div class="sk-stat shimmer" *ngFor="let st of statSkeleton"></div>
            </div>
        </div>
      </div>
    </div>
    
    <ng-container *ngIf="!matches.loading() && matches.matches() as list; else noFetchYet">
      <ng-container *ngIf="list.length; else noMatchesYet">
        <div class="matches-list">
          <div *ngFor="let match of list" class="match-card"
               [class.crush-card]="match.name === 'Your Kellogg MBA Crush'">
            <div class="match-header">
              <h3 class="match-name">{{ match.name }}</h3>
              <app-similarity-meter 
                [score]="match.score" 
                [overlap]="match.overlap">
              </app-similarity-meter>
            </div>
            
            <!-- Special message for MBA Crush -->
            <div *ngIf="match.name === 'Your Kellogg MBA Crush'" class="crush-message">
              <p class="sorry-note">🙈 Sorry, they don't have anything in common with you.</p>
            </div>
            
            <div class="match-details">
              <div class="artists-section">
                <h4 class="artists-title">Artists in Common <span class="info-icon" appTooltip="Tuples (#A/#B) show rank positions: #A = your rank, #B = their rank.">?</span></h4>
                <div class="artists-grid">
                  <span 
                    *ngFor="let artist of visibleArtists(match)" 
                    class="artist-tag"
                    [class.crush-artist]="match.name === 'Your Kellogg MBA Crush'">
                    {{ formatArtist(artist) }}
                  </span>
                </div>
                <div class="artist-actions" *ngIf="match.artists.length > artistsPerPage">
                  <button type="button" class="toggle-btn" (click)="toggleShowAll(match)">
                    {{ isShowAll(match) ? 'Show Less' : 'Show All (' + match.artists.length + ')' }}
                  </button>
                </div>
                <div class="artist-pagination" *ngIf="!isShowAll(match) && match.artists.length > artistsPerPage">
                  <button type="button" (click)="prevArtists(match)" [disabled]="artistPage(match)===0">Prev</button>
                  <span class="page-indicator">{{ artistPage(match)+1 }} / {{ totalArtistPages(match) }}</span>
                  <button type="button" (click)="nextArtists(match)" [disabled]="artistPage(match) >= totalArtistPages(match)-1">Next</button>
                </div>
              </div>
              
              <div class="match-stats">
                <div class="stat">
                  <span class="stat-label">Match Score:</span>
                  <span class="stat-value">{{ (match.score * 100) | number:'1.1-1' }}%</span>
                </div>
                <div class="stat">
                  <span class="stat-label">Shared Artists:</span>
                  <span class="stat-value">{{ match.overlap }}</span>
                </div>
                <div class="stat">
                  <span class="stat-label">Total Artists:</span>
                  <span class="stat-value">{{ match.artists.length || 0 }}</span>
                </div>
                <div class="stat" *ngIf="match.program">
                  <span class="stat-label">Program:</span>
                  <span class="stat-value">{{ match.program }}</span>
                </div>
                <div class="stat" *ngIf="match.graduationYear">
                  <span class="stat-label">Graduation Year:</span>
                  <span class="stat-value">{{ match.graduationYear }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </ng-container>
      <ng-template #noMatchesYet>
        <p class="empty-msg">No matches yet, but come back soon! We need more users to generate high-quality matches.</p>
      </ng-template>
    </ng-container>
    <ng-template #noFetchYet>
      <p *ngIf="!matches.loading()" class="empty-msg">No matches retrieved yet. Add or update your favorite artists to get started.</p>
    </ng-template>
    <div class="actions">
      <button type="button" (click)="back()">Back</button>
    </div>
  </section>
  `
})
export class MatchesComponent implements OnInit, OnDestroy {
  limitOptions = [10,20,30,40,50];
  overlapsLimitOptions = [5,10,15,20,30,40,50];
  limit = 50;
  overlapsLimit: number | null = null;
  range: 'short_term' | 'medium_term' | 'long_term' = 'medium_term';
  ranges = [
    { value: 'short_term' as const, label: 'Last 4 Weeks', tooltip: 'Short-term listening (past ~4 weeks)' },
    { value: 'medium_term' as const, label: 'Last 6 Months', tooltip: 'Medium-term listening (past ~6 months)' },
    { value: 'long_term' as const, label: 'Last Year', tooltip: 'Long-term listening (past year+)' }
  ];
  skeletonArray = Array.from({length:3});
  tagSkeleton = Array.from({length:10});
  statSkeleton = Array.from({length:4});
  // Maintain per-match pagination state keyed by match name (assuming uniqueness for UI purposes)
  private artistPages: Record<string, number> = {};
  private showAll: Record<string, boolean> = {};
  artistsPerPage = 25;
  showRankInfo = false;
  private limitDebounce: any;
  debounceMs = 400;

  constructor(public matches: MatchService, private router: Router, private spotify: SpotifyService) {
    effect(() => { if (!this.matches.matches()) { /* could redirect */ } });
    this.matches.fetchIfReady();
  }
  ngOnInit(): void {
    // URL query params override stored prefs (deep-link)
    const urlParams = new URLSearchParams(window.location.search);
    const qRange = urlParams.get('range');
    if (qRange === 'short_term' || qRange === 'medium_term' || qRange === 'long_term') {
      this.range = qRange;
    }
    const qLimit = urlParams.get('limit');
    if (qLimit) {
      const ln = parseInt(qLimit, 10);
      if (!isNaN(ln) && this.limitOptions.includes(ln)) this.limit = ln;
    }
    const qOv = urlParams.get('overlapsLimit');
    if (qOv) {
      const ov = parseInt(qOv, 10);
      if (!isNaN(ov) && ov > 0) this.overlapsLimit = ov;
    }
    const stored = localStorage.getItem('kmmMatchLimit');
    if (stored) {
      const num = parseInt(stored, 10);
      if (this.limitOptions.includes(num)) {
        this.limit = num;
        // Refetch with stored limit if different from default
        if (num !== 50) {
          this.refetch();
        }
      }
    }
    const storedDb = localStorage.getItem('kmmMatchLimitDebounceMs');
    if (storedDb) {
      const d = parseInt(storedDb, 10);
      if (!isNaN(d) && d >= 0 && d <= 3000) this.debounceMs = d;
    }
    const storedOv = localStorage.getItem('kmmOverlapsLimit');
    if (storedOv) {
      const ov = parseInt(storedOv, 10);
      if (!isNaN(ov) && ov > 0) this.overlapsLimit = ov;
    }
    const storedRange = localStorage.getItem('kmmMatchRange');
    if (storedRange === 'short_term' || storedRange === 'medium_term' || storedRange === 'long_term') {
      this.range = storedRange;
    }
  }
  refetch() { this.matches.fetch(this.range, this.limit, this.overlapsLimit); }
  private updateUrlState() {
    const params = new URLSearchParams();
    params.set('range', this.range);
    params.set('limit', String(this.limit));
    if (this.overlapsLimit) params.set('overlapsLimit', String(this.overlapsLimit));
    history.replaceState({}, '', this.router.url.split('?')[0] + '?' + params.toString());
  }
  selectRange(r: 'short_term'|'medium_term'|'long_term') {
    if (this.range === r) return;
    this.range = r;
    localStorage.setItem('kmmMatchRange', r);
    if (this.matches.loading()) {
      // defer until loading ends
      const wait = setInterval(() => {
        if (!this.matches.loading()) { clearInterval(wait); this.refetch(); }
      }, 150);
    } else {
      // small debounce to allow rapid toggling without spamming network
      if (this.limitDebounce) clearTimeout(this.limitDebounce);
      this.limitDebounce = setTimeout(() => this.refetch(), 180);
    }
  }
  rangeLabel(): string {
    switch (this.range) {
      case 'short_term': return 'last 4 weeks';
      case 'medium_term': return 'last 6 months';
      case 'long_term': return 'long-term (year+)';
    }
  }
  onLimitChange() {
    localStorage.setItem('kmmMatchLimit', this.limit.toString());
    if (this.limitDebounce) clearTimeout(this.limitDebounce);
    this.limitDebounce = setTimeout(() => { this.updateUrlState(); this.refetch(); }, this.debounceMs);
  }
  onDebounceChange() {
    if (this.debounceMs < 0) this.debounceMs = 0;
    if (this.debounceMs > 3000) this.debounceMs = 3000;
    localStorage.setItem('kmmMatchLimitDebounceMs', this.debounceMs.toString());
  }
  resetControls() {
    this.limit = 50;
    this.overlapsLimit = null;
    this.debounceMs = 400;
    localStorage.setItem('kmmMatchLimit', this.limit.toString());
    localStorage.setItem('kmmMatchLimitDebounceMs', this.debounceMs.toString());
    localStorage.removeItem('kmmOverlapsLimit');
    this.updateUrlState();
    this.refetch();
  }
  onOverlapsLimitChange() {
    if (this.overlapsLimit !== null && this.overlapsLimit <= 0) this.overlapsLimit = 5;
    if (this.overlapsLimit !== null) {
      localStorage.setItem('kmmOverlapsLimit', this.overlapsLimit.toString());
    } else {
      localStorage.removeItem('kmmOverlapsLimit');
    }
    this.updateUrlState();
    this.refetch();
  }
  back(): void { this.router.navigateByUrl('/chicago-events'); }
  async connectSpotify() { await this.spotify.beginAuth(); }

  artistPage(match: any): number { return this.artistPages[match.name] || 0; }
  totalArtistPages(match: any): number { return Math.ceil(match.artists.length / this.artistsPerPage); }
  visibleArtists(match: any): any[] {
    const source = (match.overlaps && match.overlaps.length) ? match.overlaps.map((o: any) => ({ name: o.name, anchorRank: o.anchorRank, otherRank: o.otherRank })) : match.artists;
    if (this.isShowAll(match)) return source;
    const page = this.artistPage(match);
    const start = page * this.artistsPerPage;
    return source.slice(start, start + this.artistsPerPage);
  }
  nextArtists(match: any) { const p = this.artistPage(match); if (p < this.totalArtistPages(match)-1) { this.artistPages[match.name] = p+1; } }
  prevArtists(match: any) { const p = this.artistPage(match); if (p > 0) { this.artistPages[match.name] = p-1; } }
  toggleShowAll(match: any) { this.showAll[match.name] = !this.showAll[match.name]; }
  isShowAll(match: any): boolean { return !!this.showAll[match.name]; }
  formatArtist(a: any): string {
    if (!a) return '';
    if (typeof a === 'string') return a;
    const name = a.name || '';
    if (a.anchorRank && a.otherRank) return `${name} (#${a.anchorRank}/${a.otherRank})`;
    return name;
  }
  ngOnDestroy(): void { if (this.limitDebounce) clearTimeout(this.limitDebounce); }
}
