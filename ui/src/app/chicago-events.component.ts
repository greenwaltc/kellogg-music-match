import { Component, OnInit, OnDestroy, HostListener } from '@angular/core';
import { trigger, state, style, transition, animate } from '@angular/animations';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { ConcertInterestService } from './concert-interest.service';
import { AuthService } from './auth.service';
import { Subject, BehaviorSubject, combineLatest } from 'rxjs';
import { debounceTime, distinctUntilChanged, switchMap, takeUntil, catchError } from 'rxjs/operators';
import { of } from 'rxjs';

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
    ])
  ],
  template: `
    <div class="chicago-events-container">
      <div class="page-header">
        <h1>🎵 Chicago Events</h1>
        <p class="subtitle">Discover upcoming concerts and shows in the Windy City</p>
      </div>

      <!-- Search Bar -->
      <div class="search-section">
        <div class="search-input-wrapper">
          <input 
            type="text" 
            [(ngModel)]="searchQuery" 
            (ngModelChange)="onSearchChange($event)"
            placeholder="Search by artist name..."
            class="search-input"
            [disabled]="isLoading && events.length === 0"
          >
          <span class="search-icon">🔍</span>
        </div>
        
        <div class="search-info" *ngIf="totalCount > 0">
          <span class="results-count">{{totalCount}} events found</span>
          <span class="filter-info" *ngIf="searchQuery">for "{{searchQuery}}"</span>
        </div>
      </div>

      <!-- Loading State -->
      <div *ngIf="isLoading && events.length === 0" class="loading-state">
        <div class="loading-spinner"></div>
        <p>Loading Chicago events...</p>
      </div>

      <!-- Error State -->
      <div *ngIf="error" class="error-state">
        <div class="error-icon">⚠️</div>
        <h3>Oops! Something went wrong</h3>
        <p>{{error}}</p>
        <button (click)="retry()" class="retry-button">Try Again</button>
      </div>

      <!-- Events List -->
      <div *ngIf="!isLoading || events.length > 0" class="events-section">
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
            *ngFor="let event of events; trackBy: trackByEventId" 
            class="event-card"
            [class.highlighted]="isEventHighlighted(event)"
          >
            <!-- Event Date Badge -->
            <div class="date-badge">
              <div class="date-day">{{formatDate(event.date, 'day')}}</div>
              <div class="date-month">{{formatDate(event.date, 'month')}}</div>
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
    </div>
  `,
  styles: [``] // Styles moved to global styles.css to stay under component budget
})
export class ChicagoEventsComponent implements OnInit, OnDestroy {
  events: ChicagoEvent[] = [];
  searchQuery = '';
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
  private searchSubject = new BehaviorSubject<string>('');
  private currentOffset = 0;
  private readonly pageSize = 20;
  
  private apiBaseUrl: string;

  constructor(private http: HttpClient, private interest: ConcertInterestService, private auth: AuthService) {
    this.apiBaseUrl = window.__kmmConfig?.apiBaseUrl || 'http://localhost:8080';
  }

  ngOnInit() {
    // Set up search with debouncing
    this.searchSubject.pipe(
      debounceTime(300),
      distinctUntilChanged(),
      takeUntil(this.destroy$)
    ).subscribe(query => {
      this.resetAndSearch(query);
    });

    // Initial load
    this.loadEvents(true);
  }

  ngOnDestroy() {
    this.destroy$.next();
    this.destroy$.complete();
  }

  @HostListener('window:scroll', ['$event'])
  onScroll() {
    // Infinite scroll: load more when near bottom
    const scrollPosition = window.pageYOffset + window.innerHeight;
    const documentHeight = document.documentElement.scrollHeight;
    
    if (scrollPosition > documentHeight - 1000 && this.hasMore && !this.isLoadingMore) {
      this.loadMoreEvents();
    }
  }

  onSearchChange(value: string) {
    this.searchQuery = value;
    this.searchSubject.next(value);
  }

  onTicketClick(event: Event, ticketUrl: string) {
    console.log('Ticket button clicked:', ticketUrl);
    
    // Try to open in new tab/window
    try {
      const newWindow = window.open(ticketUrl, '_blank', 'noopener,noreferrer');
      
      if (!newWindow) {
        // Popup was blocked, show a message or fallback
        console.warn('Popup blocked. Fallback: navigating in same tab');
        // Prevent default anchor behavior and navigate manually
        event.preventDefault();
        window.location.href = ticketUrl;
      } else {
        console.log('Successfully opened ticket URL in new window');
        // Let the default anchor behavior work as well
      }
    } catch (error) {
      console.error('Error opening ticket URL:', error);
      // Fallback: let the default anchor behavior handle it
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

    const query = searchQuery ?? this.searchQuery;
    if (query.trim()) {
      params.append('artistName', query.trim());
    }

    this.http.get<ChicagoEventsResponse>(`${this.apiBaseUrl}/chicago/events?${params}`)
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
        },
        error: (error) => {
          console.error('Error loading Chicago events:', error);
          this.error = 'Failed to load events. Please try again.';
          this.isLoading = false;
          this.isLoadingMore = false;
        }
      });
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
    if (this.searchQuery.trim()) params.append('artistName', this.searchQuery.trim());
    this.http.get<ChicagoEventsResponse>(`${this.apiBaseUrl}/chicago/events?${params}`)
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

  formatDate(dateString: string, format: 'day' | 'month' | 'time'): string {
    const date = new Date(dateString);
    
    switch (format) {
      case 'day':
        return date.getDate().toString();
      case 'month':
        return date.toLocaleDateString('en-US', { month: 'short' }).toUpperCase();
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
    this.http.get<ChicagoEvent>(`${this.apiBaseUrl}/chicago/events/${eventId}`)
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