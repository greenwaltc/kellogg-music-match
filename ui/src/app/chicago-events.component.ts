import { Component, OnInit, OnDestroy, HostListener, ViewChild, ElementRef, AfterViewInit, ViewChildren, QueryList } from '@angular/core';
import { trigger, state, style, transition, animate } from '@angular/animations';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { ApiBaseService } from './api-base.service';
import { ConcertInterestService } from './concert-interest.service';
import { AuthService } from './auth.service';
import { Subject } from 'rxjs';
import { takeUntil, catchError } from 'rxjs/operators';
import { of } from 'rxjs';
import { SpotifyService } from './spotify.service';

interface ChicagoEvent {
  id: string;
  name: string;
  date: string;
  venue: {
    name: string;
    address: {
      street?: string;
      city: string;
      state?: string;
      country: string;
      postal?: string;
    };
  };
  artists: Array<{
    name: string;
    genres?: string[];
  }>;
  priceRange?: {
    min: number;
    max: number;
    currency: string;
  };
  ticketUrl?: string;
  // Backend aggregated user interest buckets (new fields)
  // Deprecated ID arrays (still accepted for backward compatibility)
  interestedUserIds?: string[];
  goingUserIds?: string[];
  lookingForGroupUserIds?: string[];
  // New full name arrays
  interestedUsers?: string[];
  goingUsers?: string[];
  lookingForGroupUsers?: string[];
  myInterest?: 'INTERESTED' | 'GOING' | 'LOOKING_FOR_GROUP';
}

interface ChicagoEventsResponse {
  events: ChicagoEvent[];
  hasMore: boolean;
  totalCount: number;
}

@Component({
  selector: 'app-chicago-events',
  standalone: true,
  imports: [CommonModule, FormsModule],
  animations: [
    trigger('expandCollapse', [
      state('collapsed', style({ height: '0px', opacity: 0, padding: '0 0.5rem', overflow: 'hidden' })),
      state('expanded', style({ height: '*', opacity: 1, padding: '0.6rem 0.65rem 0.7rem', overflow: 'hidden' })),
      transition('collapsed <=> expanded', animate('220ms cubic-bezier(0.4, 0.0, 0.2, 1)')),
    ]),
    // Fade/scale for scroll-to-top button
    trigger('fadeScale', [
      state('hidden', style({ opacity: 0, transform: 'scale(.75)', pointerEvents: 'none' })),
      state('visible', style({ opacity: 1, transform: 'scale(1)' })),
      transition('hidden => visible', animate('160ms cubic-bezier(0.4,0,0.2,1)')),
      transition('visible => hidden', animate('130ms cubic-bezier(0.4,0,0.2,1)'))
    ])
  ],
  template: `
  <div class="chicago-events-container">
      <div class="page-header">
        <h1>🎵 Chicago Events</h1>
        <p class="subtitle">Discover upcoming concerts and shows in the Windy City</p>
        <!-- Spotify sync moved to Matches page -->
      </div>

      <!-- Search Bar -->
      <div class="search-section">
        <div class="search-input-wrapper">
          <input 
            #searchBox
            type="text" 
            [(ngModel)]="searchQuery" 
            (ngModelChange)="onSearchChange($event)"
            placeholder="Search by artist name..."
            class="search-input" [class.loading]="searchLoading"
            [disabled]="isLoading && events.length === 0"
            (keyup.enter)="handleEnterSearch()"
            autocomplete="off"
          >
          <span class="search-icon">🔍</span>
          <button 
            *ngIf="searchQuery" 
            type="button" 
            (click)="clearSearch()" 
            aria-label="Clear search" 
            class="inline-clear-btn"
            [disabled]="searchLoading"
          >×</button>
        </div>
        <!-- Simple filters row -->
        <div class="simple-filters" style="margin-top:.75rem; display:flex; justify-content:center; gap:1.25rem; flex-wrap:wrap; align-items:center; font-size:.8rem;">
          <label style="display:inline-flex; align-items:center; gap:.4rem; cursor:pointer; user-select:none;">
            <input type="checkbox" [(ngModel)]="onlyWithInterest" (change)="onAnyInterestToggle()" />
            <span>Only events with interested Kellogg students</span>
          </label>
          <label style="display:inline-flex; align-items:center; gap:.4rem; cursor:pointer; user-select:none; opacity: {{ isAuthenticated ? 1 : 0.6 }};">
            <input type="checkbox" [(ngModel)]="onlyMyTopArtists" (change)="onOnlyTopArtistsToggle()" [disabled]="!isAuthenticated" />
            <span>Only my top artists</span>
          </label>
          <label style="display:inline-flex; align-items:center; gap:.35rem; user-select:none;">
            <span style="opacity:.85;">Top N</span>
            <input type="number" min="1" max="200" step="1" [(ngModel)]="topNArtists" (change)="onTopNArtistsChange()" [disabled]="!onlyMyTopArtists || !isAuthenticated" style="width:68px; padding:.25rem .4rem; border-radius:6px; border:1px solid var(--color-border); background: var(--color-bg-soft); color: var(--color-text);" />
          </label>
        </div>
        <div style="margin-top:0.75rem; text-align:center; display:flex; justify-content:center; gap:.5rem; flex-wrap:wrap;">
          <button class="retry-button" (click)="performSearch()" [disabled]="!searchQuery.trim() || searchLoading || isLoadingMore" style="min-width:110px; position:relative;">
            <span *ngIf="!searchLoading">Search</span>
            <span *ngIf="searchLoading" style="display:inline-flex; align-items:center; gap:.4rem;">
              <span class="loading-spinner small" style="width:18px; height:18px; border-width:2px; margin:0;"></span> Searching
            </span>
          </button>
          <button class="retry-button" style="background:var(--color-border); color:var(--color-text);" (click)="clearSearch()" [disabled]="searchLoading || (!activeSearchQuery && !searchQuery)">Clear</button>
        </div>
        
        <div class="search-info" *ngIf="totalCount > 0">
          <span class="results-count">{{totalCount}} events found</span>
          <span class="filter-info" *ngIf="activeSearchQuery"> for "{{activeSearchQuery}}"</span>
        </div>
        <div *ngIf="!searchLoading && searchQuery.trim() && searchQuery.trim() !== activeSearchQuery" style="text-align:center; margin-top:.35rem; font-size:.7rem; color:var(--color-text-muted);">
          Press Enter or Search to update results
        </div>
      </div>

      <!-- Loading State -->
      <div *ngIf="showInitialSkeletons" class="loading-skeletons">
        <div class="skeleton-grid">
          <div class="skeleton-card" *ngFor="let sk of skeletonArray">
            <div class="sk-date"></div>
            <div class="sk-lines">
              <div class="sk-line w60"></div>
              <div class="sk-line w80"></div>
              <div class="sk-line w40"></div>
              <div class="sk-line w70"></div>
            </div>
          </div>
        </div>
      </div>

      <!-- Error State -->
      <div *ngIf="error" class="error-state">
        <div class="error-icon">⚠️</div>
        <h3>Oops! Something went wrong</h3>
        <p>{{error}}</p>
        <button (click)="retry()" class="retry-button">Try Again</button>
      </div>

      <!-- Events List -->
  <div *ngIf="!isLoading || events.length > 0" class="events-section" [class.fade-in]="fadeReady">
        <!-- No Results -->
        <div *ngIf="events.length === 0 && !isLoading && !error" class="no-results">
          <div class="no-results-icon">🎭</div>
          <h3>No events found</h3>
          <p *ngIf="searchQuery">Try adjusting your search for "{{searchQuery}}"</p>
          <p *ngIf="!searchQuery">No upcoming events are currently scheduled.</p>
        </div>

        <!-- Event Cards -->
        <div class="events-grid" *ngIf="events.length > 0">
          <div 
            *ngFor="let event of events; index as i; trackBy: trackByEventId" 
            #eventCard
            class="event-card"
            [class.highlighted]="isEventHighlighted(event)"
            [style.animationDelay]="delayVariance(event,i)"
            [class.stagger-card]="initialStagger"
          >
            <!-- Event Date Badge -->
            <div class="date-badge">
              <div class="date-day">{{formatDate(event.date, 'day')}}</div>
              <div class="date-month">{{formatDate(event.date, 'month')}}</div>
              <div class="date-year">{{formatDate(event.date, 'year')}}</div>
            </div>

            <!-- Event Content -->
            <div class="event-content">
              <div class="event-header">
                <h3 class="event-title">{{event.name}}</h3>
                <div class="event-time">{{formatDate(event.date, 'time')}}</div>
              </div>

              <!-- Venue Info -->
              <div class="venue-info">
                <div class="venue-name">📍 {{event.venue.name}}</div>
                <div class="venue-address" *ngIf="event.venue.address.street">
                  {{event.venue.address.street}}, {{event.venue.address.city}}
                </div>
              </div>

              <!-- Artists -->
              <div class="artists-section" *ngIf="event.artists.length > 0">
                <div class="artists-list">
                  <span 
                    *ngFor="let artist of event.artists; let last = last" 
                    class="artist-name"
                    [class.highlighted-artist]="isArtistHighlighted(artist.name)"
                  >
                    {{artist.name}}<span *ngIf="!last">, </span>
                  </span>
                </div>
                
                <!-- Genres -->
                <div class="genres" *ngIf="getEventGenres(event).length > 0">
                  <span 
                    *ngFor="let genre of getEventGenres(event)" 
                    class="genre-tag"
                  >
                    {{genre}}
                  </span>
                </div>
              </div>

              <!-- Ticket Info -->
              <div class="ticket-section" *ngIf="event.ticketUrl">
                <a 
                  [href]="event.ticketUrl" 
                  target="_blank" 
                  rel="noopener noreferrer"
                  class="ticket-button"
                  (click)="onTicketClick($event, event.ticketUrl)"
                  title="Open ticket purchase page: {{event.ticketUrl}}"
                >
                  Get Tickets 🎫
                </a>

                <!-- Debug info (remove in production) -->
                <div class="debug-info" style="font-size: 0.7rem; color: var(--color-text-muted); margin-top: 0.5rem;" *ngIf="false">
                  URL: {{event.ticketUrl}}
                </div>
              </div>

              <!-- Interest Actions -->
              <div class="interest-actions">
                <div class="interest-buttons">
                  <button type="button" class="interest-btn" [class.active]="event.myInterest==='INTERESTED'" (click)="setInterest(event.id, 'INTERESTED')" title="Mark as Interested" aria-label="Mark as Interested">★ Interested ({{totalInterested(event)}})</button>
                  <button type="button" class="interest-btn" [class.active]="event.myInterest==='GOING'" (click)="setInterest(event.id, 'GOING')" title="Mark as Going" aria-label="Mark as Going">✅ Going ({{totalGoing(event)}})</button>
                  <button type="button" class="interest-btn" [class.active]="event.myInterest==='LOOKING_FOR_GROUP'" (click)="setInterest(event.id, 'LOOKING_FOR_GROUP')" title="Looking For Group" aria-label="Looking For Group">👥 Looking for Group ({{totalLFG(event)}})</button>
                  <button type="button" class="interest-btn danger" [class.active]="!event.myInterest" (click)="removeInterest(event.id)" title="Clear My Interest" aria-label="Clear My Interest">✖ Clear</button>
                  <button type="button" class="toggle-attendees inline" *ngIf="hasAnyInterest(event)" (click)="toggleInterestList(event)">
                    {{ isInterestExpanded(event.id) ? 'Hide' : 'View' }} Attendees
                    <span class="chevron">{{ isInterestExpanded(event.id) ? '▴' : '▾' }}</span>
                  </button>
                </div>
                <div class="interest-attendees" [@expandCollapse]="isInterestExpanded(event.id) ? 'expanded' : 'collapsed'" *ngIf="isInterestExpanded(event.id)">
                  <div *ngIf="loadingAttendeesFor === event.id" class="loading-attendees">Loading attendees...</div>
                  <div *ngIf="!loadingAttendeesFor && !hasAnyInterest(event)" class="empty-attendees">No interest recorded yet.</div>
                  <div *ngIf="!loadingAttendeesFor && totalInterested(event) > 0" class="interest-category">
                    <div class="category-title">Interested</div>
                    <div class="names-list">
                      <span *ngFor="let name of (event.interestedUsers || [])" class="user-name">{{name}}</span>
                    </div>
                  </div>
                  <div *ngIf="!loadingAttendeesFor && totalGoing(event) > 0" class="interest-category">
                    <div class="category-title">Going</div>
                    <div class="names-list">
                      <span *ngFor="let name of (event.goingUsers || [])" class="user-name">{{name}}</span>
                    </div>
                  </div>
                  <div *ngIf="!loadingAttendeesFor && totalLFG(event) > 0" class="interest-category">
                    <div class="category-title">Looking for Group</div>
                    <div class="names-list">
                      <span *ngFor="let name of (event.lookingForGroupUsers || [])" class="user-name">{{name}}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Load More -->
        <div *ngIf="hasMore && events.length > 0" class="load-more-section">
          <button 
            (click)="loadMoreEvents()" 
            [disabled]="isLoadingMore"
            class="load-more-button"
          >
            <span *ngIf="!isLoadingMore">Load More Events</span>
            <span *ngIf="isLoadingMore">Loading...</span>
          </button>
        </div>

        <!-- Loading More Indicator -->
        <div *ngIf="isLoadingMore" class="loading-more">
          <div class="loading-spinner small"></div>
          <span>Loading more events...</span>
        </div>
      </div>
      <!-- Scroll Controls -->
      <ng-container *ngIf="!isTouchDevice; else mobileFab">
        <div class="desktop-scroll-fab" [@fadeScale]="showScrollTop ? 'visible':'hidden'" [class.has-prev]="showPrevFromHistory">
          <button
            type="button"
            class="fab-main-desktop"
            (click)="scrollToTop()"
            aria-label="Scroll to top"
            title="Scroll to top"
            [class.pulse-on-appear]="pulseScrollTop"
          >⇧</button>
          <button *ngIf="showPrevFromHistory"
            type="button"
            class="fab-sub"
            (click)="scrollToPrevious()"
            aria-label="Scroll to previous position"
            title="Return to previous position"
          >↺</button>
        </div>
      </ng-container>
      <ng-template #mobileFab>
        <div class="fab-wrapper" [@fadeScale]="showScrollTop ? 'visible':'hidden'">
          <button type="button" class="fab-main" aria-label="Scroll controls"
            (pointerdown)="onFabPointerDown($event)" (pointerup)="onFabPointerUp($event)" (pointerleave)="onFabPointerCancel()"
            (click)="onFabClick()" [class.pulse-on-appear]="pulseScrollTop"
          >⇧</button>
          <div class="fab-actions" [class.open]="radialMenuOpen">
            <button type="button" class="fab-action" (click)="scrollToTop(); closeRadialMenu()" aria-label="Scroll to top">↑</button>
            <button *ngIf="scrollHistory.length" type="button" class="fab-action second" (click)="scrollToPrevious(); closeRadialMenu()" aria-label="Scroll to previous position">↺</button>
          </div>
        </div>
      </ng-template>
    </div>
  `,
  styles: [`
    /* Desktop hover-expand group */
    .desktop-scroll-fab { position:fixed; right:1.25rem; bottom:1.25rem; z-index:1000; display:flex; align-items:center; gap:.6rem; }
    .fab-main-desktop { width:52px; height:52px; border-radius:50%; background:var(--color-accent,#2d6cdf); color:#fff; font-size:1.15rem; font-weight:600; border:none; box-shadow:0 6px 20px rgba(0,0,0,.28); cursor:pointer; display:flex; align-items:center; justify-content:center; transition:box-shadow .25s, transform .25s, background .25s; }
    .fab-main-desktop:hover { transform:translateY(-3px); }
    .fab-main-desktop:active { transform:translateY(0); box-shadow:0 3px 12px rgba(0,0,0,.35); }
    .fab-sub { width:40px; height:40px; border-radius:50%; border:none; background:var(--color-border,#444); color:#fff; font-size:.9rem; font-weight:600; box-shadow:0 4px 14px rgba(0,0,0,.25); cursor:pointer; opacity:0; transform:scale(.6) translateX(8px); transition:opacity .25s cubic-bezier(.4,0,.2,1), transform .32s cubic-bezier(.4,0,.2,1); }
    .desktop-scroll-fab:hover .fab-sub, .desktop-scroll-fab:focus-within .fab-sub { opacity:1; transform:scale(1) translateX(0); }
    .fab-sub:hover { transform:scale(1) translateX(0) translateY(-2px); }
    .fab-sub:active { transform:scale(.95) translateX(0); }
    @keyframes pulseShadow { 0% { box-shadow:0 0 0 0 rgba(45,108,223,0.55);} 60% { box-shadow:0 0 0 14px rgba(45,108,223,0);} 100% { box-shadow:0 0 0 0 rgba(45,108,223,0);} }
    .pulse-on-appear { animation: pulseShadow 1150ms ease-out 120ms 1; }
    /* Radial FAB (mobile) */
    .fab-wrapper { position: fixed; right: 1.05rem; bottom: 1.05rem; z-index: 1100; }
    .fab-main { width: 52px; height: 52px; border-radius: 50%; background: var(--color-accent,#2d6cdf); color:#fff; font-size:1.15rem; font-weight:600; border:none; box-shadow:0 6px 20px rgba(0,0,0,.28); cursor:pointer; display:flex; align-items:center; justify-content:center; transition: box-shadow .25s, transform .25s; }
    .fab-main:active { transform: scale(.94); }
    .fab-actions { position:absolute; bottom: 56px; right:4px; width:0; height:0; pointer-events:none; }
    .fab-actions.open { pointer-events: auto; }
    .fab-action { position:absolute; width:40px; height:40px; border-radius:50%; border:none; background: var(--color-border,#444); color:#fff; font-size:.85rem; font-weight:600; box-shadow:0 4px 16px rgba(0,0,0,.25); opacity:0; transform: scale(.6) translate(0,0); transition: opacity .25s cubic-bezier(.4,0,.2,1), transform .32s cubic-bezier(.4,0,.2,1); }
    .fab-actions.open .fab-action { opacity:1; transform: scale(1) translate(0,0); }
    .fab-action { bottom:0; right:0; }
    .fab-actions.open .fab-action:nth-child(1) { transform: scale(1) translate(-4px, -60px); }
    .fab-actions.open .fab-action.second { transform: scale(1) translate(-54px, -34px); }
    @media (min-width: 821px) { .fab-wrapper { display:none; } }
    @media (max-width: 820px) { .desktop-scroll-fab { display:none; } }
    @media (max-width: 640px) { .fab-wrapper { right:.7rem; bottom:.7rem; } .fab-main { width:48px; height:48px; } }
  `] // Minimal local styles for the floating button; rest remain in global stylesheet
})
export class ChicagoEventsComponent implements OnInit, OnDestroy, AfterViewInit {
  events: ChicagoEvent[] = [];
  // pending (typed but not yet executed) search text
  searchQuery = '';
  // last executed search text actually applied to results
  activeSearchQuery = '';
  // indicates a user-initiated search request is in flight
  searchLoading = false;
  private pendingRestoreScrollY: number | null = null;
  private lastEnterPress = 0;
  fadeReady = false; // controls fade-in animation of results
  initialStagger = true; // only apply card staggering on very first page load (public for template)
  skeletonArray: any[] = [];
  // Flag to know initial request in-flight for skeletons
  get showInitialSkeletons(): boolean { return this.isLoading && !this.firstLoadCompleted && this.events.length === 0; }
  private firstLoadCompleted = false;
  private readonly SEARCH_STORAGE_KEY = 'kmm_chi_events_search';
  @ViewChild('searchBox') searchInput!: ElementRef<HTMLInputElement>;
  @ViewChildren('eventCard') eventCardElems!: QueryList<ElementRef<HTMLElement>>;
  private io?: IntersectionObserver;
  isLoading = false;
  isLoadingMore = false;
  hasMore = false;
  totalCount = 0;
  error: string | null = null;
  private expandedInterestEvents = new Set<string>();
  // For single-expanded mode & lazy load state
  private singleExpandedEventId: string | null = null;
  loadingAttendeesFor: string | null = null;
  private toggleCooldown = false;
  private lastExpandedEventId: string | null = null;
  
  private destroy$ = new Subject<void>();
  private currentOffset = 0;
  private readonly pageSize = 20;
  
  // New filter flag
  onlyWithInterest = false;
  private ANY_INTEREST_STORAGE_KEY = 'kmm_chi_any_interest';
  // New: Only my top artists filter + Top N
  onlyMyTopArtists = false;
  topNArtists = 20;
  private ONLY_TOP_STORAGE_KEY = 'kmm_chi_only_top_artists';
  private TOPN_STORAGE_KEY = 'kmm_chi_top_n_artists';
  // Observer scroll throttling
  private ioPaused = false;
  private scrollLastY = 0;
  private scrollLastTime = 0;
  private ioResumeTimer: any;
  private ioCurrentThreshold = 0.15;
  private dynamicThresholdAdjusted = false;
  // Scroll-to-top visibility flag
  showScrollTop = false;
  // IntersectionObserver for determining when top content is off-screen
  private scrollTopObserver?: IntersectionObserver;
  // Session scroll restoration
  private readonly SCROLL_POS_KEY = 'kmm_chi_scrollY';
  private sessionRestoreScrollY: number | null = null;
  private lastPersistScrollTime = 0;
  // Scroll history stack & UI state for previous position toggle
  scrollHistory: number[] = [];
  private readonly MAX_SCROLL_HISTORY = 5;
  showPrevFromHistory = false;
  pulseScrollTop = false;
  // Spotify sync state
  spotifySyncing = false;
  spotifyStatusMessage: string | null = null;

  // API base service (replaces legacy hardcoded base URL)
  // Always use this.api.url(path) or this.api.baseUrl

  startSpotifySync() {
    if (this.spotifySyncing) { return; }
    this.spotifySyncing = true;
    this.spotifyStatusMessage = 'Redirecting to Spotify...';
    this.spotifyService.beginAuth().catch(err => {
      this.spotifyStatusMessage = 'Failed to start Spotify auth';
      this.spotifySyncing = false;
      console.error('Spotify auth error', err);
    });
  }
  private hideDelayTimer: any;
  private pendingHide = false;
  // Touch / radial menu state
  isTouchDevice = false;
  radialMenuOpen = false;
  private longPressTimer: any;
  private longPressActivated = false;
  private readonly LONG_PRESS_THRESHOLD = 450; // ms
  private readonly SCROLL_HISTORY_KEY = 'kmm_chi_scroll_history';
  private lastHistoryPushTime = 0; // performance.now() timestamp of last accepted history entry

  constructor(private http: HttpClient, private interest: ConcertInterestService, private auth: AuthService, private spotifyService: SpotifyService, private api: ApiBaseService) {
    // Restore anyInterest flag
    try {
      const storedInterest = localStorage.getItem(this.ANY_INTEREST_STORAGE_KEY);
      if (storedInterest === 'true') this.onlyWithInterest = true;
    } catch {}

    // Restore Only my top artists
    try {
      const storedOnlyTop = localStorage.getItem(this.ONLY_TOP_STORAGE_KEY);
      if (storedOnlyTop === 'true') this.onlyMyTopArtists = true;
    } catch {}
    // Restore Top N value
    try {
      const rawN = localStorage.getItem(this.TOPN_STORAGE_KEY);
      const n = rawN ? parseInt(rawN, 10) : NaN;
      if (!isNaN(n) && n >= 1 && n <= 50) this.topNArtists = n;
    } catch {}
  }

  ngOnInit() {
    // Touch detection
    try { this.isTouchDevice = (('ontouchstart' in window) || (navigator.maxTouchPoints || 0) > 0); } catch { this.isTouchDevice = false; }
    this.computeSkeletons();
    // Restore persisted search (manual search paradigm)
    this.fadeReady = false;
    // Capture prior scroll position from session (used if not doing an in-page search restore)
    try {
      const raw = sessionStorage.getItem(this.SCROLL_POS_KEY);
      if (raw) {
        const val = parseInt(raw, 10);
        if (!isNaN(val) && val > 0) this.sessionRestoreScrollY = val;
      }
    } catch { /* ignore */ }
    try {
      const hist = sessionStorage.getItem(this.SCROLL_HISTORY_KEY);
      if (hist) {
        const arr = JSON.parse(hist);
        if (Array.isArray(arr)) this.scrollHistory = arr.filter(v => typeof v === 'number').slice(-this.MAX_SCROLL_HISTORY);
      }
    } catch {}
    try {
      const saved = localStorage.getItem(this.SEARCH_STORAGE_KEY);
      if (saved && saved.trim()) {
        this.searchQuery = saved;
        this.activeSearchQuery = saved;
        this.loadEvents(true, saved);
        return;
      }
    } catch {}
    this.loadEvents(true);
  }

  ngAfterViewInit(): void {
    this.setupCardObserver();
    // Re-attach observer when the list of cards changes (pagination or search)
    this.eventCardElems.changes.subscribe(() => {
      this.setupCardObserver();
      this.setupScrollTopObserver();
    });
    // Initial setup for scroll-top visibility observer
    this.setupScrollTopObserver();
  }

  ngOnDestroy() {
    this.destroy$.next();
    this.destroy$.complete();
    if (this.io) this.io.disconnect();
    if (this.ioResumeTimer) clearTimeout(this.ioResumeTimer);
    if (this.scrollTopObserver) this.scrollTopObserver.disconnect();
  }

  @HostListener('window:scroll', ['$event'])
  onScroll() {
    // Infinite scroll: load more when near bottom
    const scrollPosition = window.pageYOffset + window.innerHeight;
    const documentHeight = document.documentElement.scrollHeight;
    
    if (scrollPosition > documentHeight - 1000 && this.hasMore && !this.isLoadingMore) {
      this.loadMoreEvents();
    }

    // Scroll velocity measurement
    const now = performance.now();
    if (this.scrollLastTime === 0) {
      this.scrollLastTime = now;
      this.scrollLastY = window.scrollY;
      return;
    }
    const dy = Math.abs(window.scrollY - this.scrollLastY);
    const dt = now - this.scrollLastTime || 1;
    const velocity = dy / dt; // px per ms
    this.scrollLastY = window.scrollY;
    this.scrollLastTime = now;
    const FAST_THRESHOLD = 2; // >2 px/ms considered fast
    if (velocity > FAST_THRESHOLD && !this.ioPaused) {
      this.pauseObserver();
    } else if (velocity <= FAST_THRESHOLD && this.ioPaused) {
      // Debounce resume
      if (this.ioResumeTimer) clearTimeout(this.ioResumeTimer);
      this.ioResumeTimer = setTimeout(() => {
        if (this.ioPaused) this.resumeObserver();
      }, 160);
    }
    // Persist scroll position (throttled) for session restore
    const nowPersist = performance.now();
    if (nowPersist - this.lastPersistScrollTime > 300) {
      this.lastPersistScrollTime = nowPersist;
      try { sessionStorage.setItem(this.SCROLL_POS_KEY, String(window.scrollY)); } catch {}
      this.captureScrollHistory(window.scrollY);
    }
    if (window.scrollY < 60) this.showPrevFromHistory = this.scrollHistory.length > 0; else if (this.showPrevFromHistory) this.showPrevFromHistory = false;
    if (this.radialMenuOpen) this.closeRadialMenu();
  }

  @HostListener('window:resize')
  onResize() {
    if (!this.firstLoadCompleted) {
      this.computeSkeletons();
    }
    // Recreate observer with new threshold if height drastically changed
    this.setupCardObserver(true);
  }

  onSearchChange(value: string) {
    this.searchQuery = value; // only update local buffer; no auto-search
  }

  performSearch() {
    this.pendingRestoreScrollY = window.scrollY;
    this.fadeReady = false;
    this.activeSearchQuery = this.searchQuery.trim();
    this.searchLoading = true;
    if (this.activeSearchQuery) {
      try { localStorage.setItem(this.SEARCH_STORAGE_KEY, this.activeSearchQuery); } catch {}
    } else {
      try { localStorage.removeItem(this.SEARCH_STORAGE_KEY); } catch {}
    }
    this.resetAndSearch(this.activeSearchQuery);
  }

  clearSearch() {
    if (!this.searchQuery && !this.activeSearchQuery) return;
    this.pendingRestoreScrollY = 0;
    this.fadeReady = false;
    this.searchQuery = '';
    if (this.activeSearchQuery !== '') {
      this.activeSearchQuery = '';
      this.resetAndSearch('');
    }
    try { localStorage.removeItem(this.SEARCH_STORAGE_KEY); } catch {}
    // Focus input after clearing (next tick to allow DOM update)
    setTimeout(() => { try { this.searchInput?.nativeElement.focus(); } catch {} }, 0);
  }

  handleEnterSearch() {
    const now = Date.now();
    if (now - this.lastEnterPress < 350) {
      this.lastEnterPress = now;
      return; // debounce rapid enter presses
    }
    this.lastEnterPress = now;
    if (!this.searchLoading && this.searchQuery.trim() && this.searchQuery.trim() !== this.activeSearchQuery) {
      this.performSearch();
    }
  }

  onTicketClick(event: Event, ticketUrl: string) {
    // Ensure only ONE navigation: prevent default anchor behavior (which would also open due to target _blank)
    if (event && typeof event.preventDefault === 'function') event.preventDefault();
    try {
      const w = window.open(ticketUrl, '_blank', 'noopener,noreferrer');
      if (!w) {
        // Popup blocked; graceful fallback to same-tab navigation
        window.location.href = ticketUrl;
      }
    } catch {
      // As a last resort navigate in current tab
      window.location.href = ticketUrl;
    }
  }



  private resetAndSearch(query: string) {
    this.currentOffset = 0;
    this.events = [];
    this.loadEvents(true, query);
  }

  private loadEvents(isInitialLoad = false, searchQuery?: string) {
    if (isInitialLoad) {
      this.isLoading = true;
      this.error = null;
    } else {
      this.isLoadingMore = true;
    }

    const params = new URLSearchParams({
      limit: this.pageSize.toString(),
      offset: this.currentOffset.toString()
    });

    if (this.onlyWithInterest) {
      params.append('anyInterest', 'true');
    }

    if (this.onlyMyTopArtists && this.isAuthenticated) {
      params.append('onlyMyTopArtists', 'true');
      if (this.topNArtists && this.topNArtists > 0) {
        params.append('topNArtists', String(Math.max(1, Math.min(200, this.topNArtists))));
      }
    }

  const query = searchQuery ?? this.activeSearchQuery;
    if (query.trim()) {
      params.append('artistName', query.trim());
    }

  this.http.get<ChicagoEventsResponse>(this.api.url(`/chicago/events?${params}`))
      .pipe(
        catchError(error => {
          console.error('Error loading Chicago events:', error);
          return of({ events: [], hasMore: false, totalCount: 0 });
        }),
        takeUntil(this.destroy$)
      )
      .subscribe({
        next: (response) => {
          if (isInitialLoad) {
            this.events = response.events;
          } else {
            this.events = [...this.events, ...response.events];
          }
          
          this.hasMore = response.hasMore;
          this.totalCount = response.totalCount;
          this.currentOffset += response.events.length;
          
          this.isLoading = false;
          this.isLoadingMore = false;
          this.error = null;
          this.searchLoading = false;
          if (isInitialLoad && this.pendingRestoreScrollY !== null) {
            const target = this.pendingRestoreScrollY;
            requestAnimationFrame(() => {
              window.scrollTo({ top: Math.min(target, document.documentElement.scrollHeight - window.innerHeight), behavior: 'instant' as ScrollBehavior });
            });
            this.pendingRestoreScrollY = null;
          }
          if (isInitialLoad) {
            requestAnimationFrame(() => { this.fadeReady = true; });
            // Mark first load complete (for skeletons) and turn off future staggering after initial animation cycle
            if (!this.firstLoadCompleted) {
              this.firstLoadCompleted = true;
              setTimeout(() => { this.initialStagger = false; }, 1000); // allow stagger animation to finish
            }
            // Re-run observer for freshly loaded cards
            this.setupCardObserver(true);
            this.setupScrollTopObserver();
            // Session-based scroll restore (only if no search restoration pending)
            if (this.pendingRestoreScrollY === null && this.sessionRestoreScrollY !== null) {
              const target = this.sessionRestoreScrollY;
              this.sessionRestoreScrollY = null; // one-time
              requestAnimationFrame(() => {
                window.scrollTo({ top: Math.min(target, document.documentElement.scrollHeight - window.innerHeight), behavior: 'instant' as ScrollBehavior });
              });
            }
          } else {
            // Do not toggle fadeReady during pagination to avoid flicker
            this.setupCardObserver();
            this.setupScrollTopObserver();
          }
        },
        error: (error) => {
          console.error('Error loading Chicago events:', error);
          this.error = 'Failed to load events. Please try again.';
          this.isLoading = false;
          this.isLoadingMore = false;
          this.searchLoading = false;
          this.pendingRestoreScrollY = null;
          // Ensure fade still becomes visible to avoid stuck opacity
          this.fadeReady = true;
          if (!this.firstLoadCompleted) this.firstLoadCompleted = true;
          this.initialStagger = false;
        }
      });
  }

  // Observer for first event card to drive scroll-top button visibility dynamically based on layout
  private setupScrollTopObserver() {
    if (typeof window === 'undefined' || !this.eventCardElems || this.eventCardElems.length === 0) {
      return;
    }
    const first = this.eventCardElems.first?.nativeElement;
    if (!first) return;
    if (this.scrollTopObserver) {
      this.scrollTopObserver.disconnect();
    }
    this.scrollTopObserver = new IntersectionObserver(entries => {
      const entry = entries[0];
      const shouldShow = !entry.isIntersecting;
      if (shouldShow) {
        if (this.hideDelayTimer) { clearTimeout(this.hideDelayTimer); this.hideDelayTimer = null; }
        this.pendingHide = false;
        const wasHidden = !this.showScrollTop;
        this.showScrollTop = true;
        if (wasHidden) {
          this.pulseScrollTop = true;
          setTimeout(() => { this.pulseScrollTop = false; }, 1400);
        }
      } else {
        if (!this.pendingHide && this.showScrollTop) {
          this.pendingHide = true;
          this.hideDelayTimer = setTimeout(() => {
            this.showScrollTop = false;
            this.pendingHide = false;
          }, 260);
        }
      }
    }, { root: null, threshold: 0.01 });
    this.scrollTopObserver.observe(first);
  }

  private captureScrollHistory(y: number) {
    if (y < 140) return; // ignore near-top trivial positions
    const now = performance.now ? performance.now() : Date.now();
    const last = this.scrollHistory[this.scrollHistory.length - 1];
    const secondLast = this.scrollHistory[this.scrollHistory.length - 2];
    const deltaToLast = last !== undefined ? Math.abs(y - last) : Infinity;
    const timeDelta = now - this.lastHistoryPushTime;

    // Base dedup: very small spatial change (<400px) from last entry -> skip
    if (deltaToLast < 400) return;

    // Time-window aggressive dedup: if within 1.5s and movement under 600px treat as noise
    if (deltaToLast < 600 && timeDelta < 1500) return;

    // Cluster compression: if last and secondLast are close (forming a cluster) and new point is far
    if (secondLast !== undefined) {
      const clusterSpan = Math.abs(last - secondLast);
      const distFromSecond = Math.abs(y - secondLast);
      // Replace the last entry instead of pushing another if cluster is small and new jump is large
      if (clusterSpan < 350 && distFromSecond > 900) {
        this.scrollHistory[this.scrollHistory.length - 1] = y;
        this.lastHistoryPushTime = now;
        this.persistScrollHistory();
        return;
      }
    }

    // Normal push path
    this.scrollHistory.push(y);
    if (this.scrollHistory.length > this.MAX_SCROLL_HISTORY) this.scrollHistory.shift();
    this.lastHistoryPushTime = now;
    this.persistScrollHistory();
  }

  private setupCardObserver(forceRecreate = false) {
    if (typeof window === 'undefined' || !('IntersectionObserver' in window)) return;
    // Respect reduced motion: if user prefers reduced motion, reveal all immediately and skip observer
    const prefersReduced = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (prefersReduced) {
      this.eventCardElems?.forEach(ref => ref.nativeElement.classList.add('in-view'));
      return;
    }
    // Disconnect previous if forced
    if (forceRecreate && this.io) {
      this.io.disconnect();
      this.io = undefined;
      this.dynamicThresholdAdjusted = false;
    }
    if (!this.io) {
      const vh = window.innerHeight || 800;
      const threshold = vh > 1200 ? 0.05 : 0.15;
      this.ioCurrentThreshold = threshold;
      this.io = new IntersectionObserver(entries => {
      entries.forEach(entry => {
        if (entry.isIntersecting) {
          entry.target.classList.add('in-view');
          // Once revealed, unobserve to avoid repeated class toggling
          this.io?.unobserve(entry.target as Element);
        }
      });
      // Dynamic threshold adjustment using rootBounds for very tall viewports
      if (!this.dynamicThresholdAdjusted && entries.length) {
        const rootH = entries[0].rootBounds?.height;
        if (rootH && rootH > 1400 && this.ioCurrentThreshold > 0.08) {
          this.dynamicThresholdAdjusted = true;
          const remaining: HTMLElement[] = [];
          this.eventCardElems?.forEach(ref => { if (!ref.nativeElement.classList.contains('in-view')) remaining.push(ref.nativeElement); });
          this.io?.disconnect();
          this.ioCurrentThreshold = 0.05;
          this.io = new IntersectionObserver(innerEntries => {
            innerEntries.forEach(inner => {
              if (inner.isIntersecting) {
                inner.target.classList.add('in-view');
                this.io?.unobserve(inner.target as Element);
              }
            });
          }, { root: null, rootMargin: '0px 0px -5% 0px', threshold: this.ioCurrentThreshold });
          remaining.forEach(el => this.io?.observe(el));
        }
      }
    }, { root: null, rootMargin: '0px 0px -5% 0px', threshold });
    }

    // Observe each card that has not yet been revealed
    this.eventCardElems?.forEach(ref => {
      const el = ref.nativeElement;
      if (!el.classList.contains('in-view')) {
        this.io?.observe(el);
      }
    });
  }

  private pauseObserver() {
    if (this.io && !this.ioPaused) {
      this.ioPaused = true;
      this.io.disconnect();
    }
  }

  private resumeObserver() {
    if (this.ioPaused) {
      this.ioPaused = false;
      // Force recreate to ensure all yet-unseen cards are reattached
      this.setupCardObserver(true);
    }
  }

  // Compute skeleton cards count based on viewport size & approximate card dimensions
  private computeSkeletons() {
    const cardMinWidth = 350; // matches grid min width
    const approxCardHeight = 210; // include padding & margin guess
    const columns = Math.max(1, Math.floor((window.innerWidth - 40) / cardMinWidth));
    const rowsNeeded = Math.max(2, Math.ceil(window.innerHeight / approxCardHeight));
    const total = Math.min(12, columns * rowsNeeded);
    this.skeletonArray = Array.from({ length: total });
  }

  // Stable small random variance derived from event id for organic delay
  delayVariance(event: ChicagoEvent, index: number): string | null {
    if (!this.initialStagger) return null;
    const id = event.id || index.toString();
    let hash = 0;
    for (let i = 0; i < id.length; i++) {
      hash = ((hash << 5) - hash) + id.charCodeAt(i);
      hash |= 0; // 32-bit
    }
    const variance = (hash % 61) - 30; // -30..30
    const base = index * 55;
    const ms = Math.max(0, base + variance);
    return ms + 'ms';
  }

  loadMoreEvents() {
    if (!this.hasMore || this.isLoadingMore) return;
    this.loadEvents(false);
  }

  retry() {
    this.currentOffset = 0;
    this.events = [];
    this.loadEvents(true);
  }

  trackByEventId(index: number, event: ChicagoEvent): string {
    return event.id;
  }

  setInterest(eventId: string, interestType: 'INTERESTED' | 'GOING' | 'LOOKING_FOR_GROUP') {
    const idx = this.events.findIndex(e => e.id === eventId);
    if (idx === -1) return;
    const previous = deepCloneEvent(this.events[idx]);
    const user = this.auth.user();
    if (user?.id || (user?.firstName && user?.lastName)) {
      const fullName = `${user.firstName} ${user.lastName}`.trim();
      this.events[idx] = this.applyLocalInterest(this.events[idx], user.id, fullName, interestType);
    }
    this.interest.setInterest(eventId, interestType).subscribe({
      next: () => this.refreshEventInterest(eventId, () => this.reexpandIfNeeded(eventId)),
      error: err => {
        console.warn('Failed to set interest', err);
        this.events[idx] = previous; // rollback
      }
    });
  }

  removeInterest(eventId: string) {
    const idx = this.events.findIndex(e => e.id === eventId);
    if (idx === -1) return;
    const previous = deepCloneEvent(this.events[idx]);
    const user = this.auth.user();
    if (user?.id || (user?.firstName && user?.lastName)) {
      const fullName = `${user.firstName} ${user.lastName}`.trim();
      this.events[idx] = this.applyLocalInterest(this.events[idx], user.id, fullName, null);
    }
    this.interest.removeInterest(eventId).subscribe({
      next: () => this.refreshEventInterest(eventId, () => this.reexpandIfNeeded(eventId)),
      error: err => {
        console.warn('Failed to remove interest', err);
        this.events[idx] = previous;
      }
    });
  }

  private refreshEventInterest(eventId: string, done?: () => void) {
    const params = new URLSearchParams({ limit: this.pageSize.toString(), offset: '0' });
    if (this.activeSearchQuery.trim()) params.append('artistName', this.activeSearchQuery.trim());
    if (this.onlyWithInterest) params.append('anyInterest', 'true');
    if (this.onlyMyTopArtists && this.isAuthenticated) {
      params.append('onlyMyTopArtists', 'true');
      if (this.topNArtists && this.topNArtists > 0) params.append('topNArtists', String(Math.max(1, Math.min(200, this.topNArtists))));
    }
  this.http.get<ChicagoEventsResponse>(this.api.url(`/chicago/events?${params}`))
      .pipe(catchError(() => of(null)))
      .subscribe(resp => {
        if (!resp) return;
        const updated = resp.events.find(e => e.id === eventId);
        if (updated) {
          const idx = this.events.findIndex(e => e.id === eventId);
          if (idx >= 0) this.events[idx] = { ...this.events[idx], ...updated };
        }
        if (done) done();
      });
  }

  onAnyInterestToggle() {
    try {
      if (this.onlyWithInterest) localStorage.setItem(this.ANY_INTEREST_STORAGE_KEY, 'true');
      else localStorage.removeItem(this.ANY_INTEREST_STORAGE_KEY);
    } catch {}
    // Reset list with new filter
    this.resetAndSearch(this.activeSearchQuery);
  }

  onOnlyTopArtistsToggle() {
    try {
      if (this.onlyMyTopArtists) localStorage.setItem(this.ONLY_TOP_STORAGE_KEY, 'true');
      else localStorage.removeItem(this.ONLY_TOP_STORAGE_KEY);
    } catch {}
    this.resetAndSearch(this.activeSearchQuery);
  }

  onTopNArtistsChange() {
    // Clamp and persist
    let n = Number(this.topNArtists) || 20;
  n = Math.max(1, Math.min(200, n));
    this.topNArtists = n;
    try { localStorage.setItem(this.TOPN_STORAGE_KEY, String(n)); } catch {}
    if (this.onlyMyTopArtists && this.isAuthenticated) {
      this.resetAndSearch(this.activeSearchQuery);
    }
  }

  get isAuthenticated(): boolean {
    const u = this.auth.user();
    return !!u && !!(u.id || u.email);
  }

  formatDate(dateString: string, format: 'day' | 'month' | 'time' | 'year'): string {
    const date = new Date(dateString);
    
    switch (format) {
      case 'day':
        return date.getDate().toString();
      case 'month':
        return date.toLocaleDateString('en-US', { month: 'short' }).toUpperCase();
      case 'year':
        return date.getFullYear().toString();
      case 'time':
        return date.toLocaleTimeString('en-US', { 
          hour: 'numeric', 
          minute: '2-digit',
          hour12: true 
        });
      default:
        return date.toLocaleDateString();
    }
  }

  getEventGenres(event: ChicagoEvent): string[] {
    const genres = new Set<string>();
    event.artists.forEach(artist => {
      artist.genres?.forEach(genre => genres.add(genre));
    });
    return Array.from(genres);
  }

  isEventHighlighted(event: ChicagoEvent): boolean {
    if (!this.searchQuery.trim()) return false;
    
    const query = this.searchQuery.toLowerCase();
    return event.artists.some(artist => 
      artist.name.toLowerCase().includes(query)
    );
  }

  isArtistHighlighted(artistName: string): boolean {
    if (!this.searchQuery.trim()) return false;
    
    const query = this.searchQuery.toLowerCase();
    return artistName.toLowerCase().includes(query);
  }

  // Interest helper methods for new UI
  hasAnyInterest(event: ChicagoEvent): boolean {
    return this.totalInterested(event) + this.totalGoing(event) + this.totalLFG(event) > 0;
  }

  totalInterested(event: ChicagoEvent): number {
    return (event.interestedUsers?.length ?? event.interestedUserIds?.length ?? 0);
  }

  totalGoing(event: ChicagoEvent): number {
    return (event.goingUsers?.length ?? event.goingUserIds?.length ?? 0);
  }

  totalLFG(event: ChicagoEvent): number {
    return (event.lookingForGroupUsers?.length ?? event.lookingForGroupUserIds?.length ?? 0);
  }

  // Updated: single expansion & lazy load via dedicated endpoint with debounce
  toggleInterestList(event: ChicagoEvent) {
    const eventId = event.id;
    if (this.toggleCooldown) return;
    this.toggleCooldown = true;
    setTimeout(() => { this.toggleCooldown = false; }, 300);

    if (this.singleExpandedEventId === eventId) {
      this.singleExpandedEventId = null;
      return;
    }
    this.singleExpandedEventId = eventId;
    this.lastExpandedEventId = eventId;
    this.loadingAttendeesFor = eventId;
  this.http.get<ChicagoEvent>(this.api.url(`/chicago/events/${eventId}`))
      .pipe(catchError(err => { console.warn('single event fetch failed', err); return of(null); }))
      .subscribe(resp => {
        if (resp) {
          const idx = this.events.findIndex(e => e.id === eventId);
          if (idx >= 0) this.events[idx] = { ...this.events[idx], ...resp };
        }
        this.loadingAttendeesFor = null;
      });
  }

  isInterestExpanded(eventId: string): boolean {
    return this.singleExpandedEventId === eventId;
  }

  private reexpandIfNeeded(eventId: string) {
    if (this.lastExpandedEventId === eventId && this.singleExpandedEventId === eventId) {
      // keep expanded
    }
  }

  // Smoothly scroll back to the top and optionally refocus the search bar
  scrollToTop() {
    try {
      const current = window.scrollY;
      if (current > 0) {
        if (this.scrollHistory[this.scrollHistory.length - 1] !== current) {
          this.scrollHistory.push(current);
          if (this.scrollHistory.length > this.MAX_SCROLL_HISTORY) this.scrollHistory.shift();
        }
      }
      window.scrollTo({ top: 0, behavior: 'smooth' });
  this.persistScrollHistory();
      // After scroll ends (approx), focus search input if present
      setTimeout(() => { this.searchInput?.nativeElement?.focus({ preventScroll: true }); }, 550);
    } catch {
      // Fallback instant scroll
      window.scrollTo(0,0);
    }
    setTimeout(() => { if (window.scrollY < 60 && this.scrollHistory.length > 0) this.showPrevFromHistory = true; }, 600);
  }

  scrollToPrevious() {
    if (this.scrollHistory.length === 0) return;
    const target = this.scrollHistory.pop();
    if (target == null) return;
    try {
      window.scrollTo({ top: target, behavior: 'smooth' });
    } catch {
      window.scrollTo(0, target);
    }
    if (this.scrollHistory.length === 0) setTimeout(() => { if (window.scrollY < 60) this.showPrevFromHistory = false; }, 700);
    this.persistScrollHistory();
  }

  onFabPointerDown(e: PointerEvent) { if (!this.isTouchDevice) return; this.longPressActivated = false; if (this.longPressTimer) clearTimeout(this.longPressTimer); this.longPressTimer = setTimeout(() => { this.longPressActivated = true; this.openRadialMenu(); }, this.LONG_PRESS_THRESHOLD); }
  onFabPointerUp(e: PointerEvent) { if (!this.isTouchDevice) return; if (this.longPressTimer) clearTimeout(this.longPressTimer); }
  onFabPointerCancel() { if (!this.isTouchDevice) return; if (this.longPressTimer) clearTimeout(this.longPressTimer); }
  onFabClick() { if (!this.isTouchDevice) { this.scrollToTop(); return; } if (this.longPressActivated) return; this.scrollToTop(); }
  openRadialMenu() { this.radialMenuOpen = true; }
  closeRadialMenu() { this.radialMenuOpen = false; this.longPressActivated = false; }
  private persistScrollHistory() { try { sessionStorage.setItem(this.SCROLL_HISTORY_KEY, JSON.stringify(this.scrollHistory)); } catch {} }
}

// Helper to deep clone an event (limited fields relevant to mutation)
function deepCloneEvent(e: ChicagoEvent): ChicagoEvent {
  return JSON.parse(JSON.stringify(e));
}

// Extend component prototype via declaration merging for helper method
// (Kept inside same file to avoid creating separate utils.)
export interface ChicagoEventsComponent {
  applyLocalInterest(event: ChicagoEvent, userId: string | undefined, fullName: string | undefined, status: 'INTERESTED' | 'GOING' | 'LOOKING_FOR_GROUP' | null): ChicagoEvent;
}

ChicagoEventsComponent.prototype.applyLocalInterest = function(event: ChicagoEvent, userId: string | undefined, fullName: string | undefined, status: 'INTERESTED' | 'GOING' | 'LOOKING_FOR_GROUP' | null): ChicagoEvent {
  // Clone to avoid mutating original references unexpectedly
  const updated: ChicagoEvent = { ...event };
  const removeId = (arr?: string[]) => (userId ? (arr || []).filter(v => v !== userId) : (arr || []));
  const removeName = (arr?: string[]) => (fullName ? (arr || []).filter(v => v !== fullName) : (arr || []));

  // Always remove from all buckets to enforce exclusivity
  updated.interestedUserIds = removeId(updated.interestedUserIds);
  updated.goingUserIds = removeId(updated.goingUserIds);
  updated.lookingForGroupUserIds = removeId(updated.lookingForGroupUserIds);
  updated.interestedUsers = removeName(updated.interestedUsers);
  updated.goingUsers = removeName(updated.goingUsers);
  updated.lookingForGroupUsers = removeName(updated.lookingForGroupUsers);

  if (status) {
    // Add to the chosen bucket
    switch (status) {
      case 'INTERESTED':
        if (userId) updated.interestedUserIds = [...(updated.interestedUserIds || []), userId];
        if (fullName) updated.interestedUsers = [...(updated.interestedUsers || []), fullName];
        break;
      case 'GOING':
        if (userId) updated.goingUserIds = [...(updated.goingUserIds || []), userId];
        if (fullName) updated.goingUsers = [...(updated.goingUsers || []), fullName];
        break;
      case 'LOOKING_FOR_GROUP':
        if (userId) updated.lookingForGroupUserIds = [...(updated.lookingForGroupUserIds || []), userId];
        if (fullName) updated.lookingForGroupUsers = [...(updated.lookingForGroupUsers || []), fullName];
        break;
    }
    (updated as any).myInterest = status;
  } else {
    delete (updated as any).myInterest;
  }
  return updated;
};