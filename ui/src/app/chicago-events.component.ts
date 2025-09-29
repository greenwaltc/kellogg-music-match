import { Component, OnInit, OnDestroy, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
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
  styles: [`
    .chicago-events-container {
      max-width: 960px;
      margin: 0 auto;
      padding: 2rem 1rem 5rem;
    }

    .page-header {
      text-align: center;
      margin-bottom: 2rem;
    }

    .page-header h1 {
      margin: 0 0 0.5rem 0;
      font-size: 2rem;
      font-weight: 700;
      background: var(--color-gradient);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
    }

    .subtitle {
      margin: 0;
      color: var(--color-text-muted);
      font-size: 1rem;
    }

    .search-section {
      background: var(--color-surface);
      border: 1px solid var(--color-border);
      border-radius: 12px;
      padding: 1.5rem;
      margin-bottom: 2rem;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    }

    .search-input-wrapper {
      position: relative;
      max-width: 500px;
      margin: 0 auto;
    }

    .search-input {
      width: 100%;
      padding: 0.65rem 0.75rem 0.65rem 2.5rem;
      border: 1px solid var(--color-border);
      border-radius: 8px;
      background: var(--color-input-bg);
      color: var(--color-text);
      font-size: 0.95rem;
      outline: none;
      transition: all 0.25s ease;
      box-sizing: border-box;
    }

    .search-input:focus {
      outline: 2px solid var(--color-accent);
      border-color: var(--color-accent);
    }

    .search-input:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }

    .search-icon {
      position: absolute;
      left: 0.75rem;
      top: 50%;
      transform: translateY(-50%);
      font-size: 1rem;
      color: var(--color-text-muted);
    }

    .search-info {
      text-align: center;
      margin-top: 1rem;
      color: var(--color-text-muted);
      font-size: 0.9rem;
    }

    .results-count {
      font-weight: 600;
      color: var(--color-accent);
    }

    .filter-info {
      margin-left: 0.5rem;
    }

    .loading-state, .error-state, .no-results {
      text-align: center;
      padding: 3rem 2rem;
      background: var(--color-surface);
      border: 1px solid var(--color-border);
      border-radius: 12px;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    }

    .loading-spinner {
      width: 50px;
      height: 50px;
      border: 4px solid var(--color-border);
      border-top: 4px solid var(--color-accent);
      border-radius: 50%;
      animation: spin 1s linear infinite;
      margin: 0 auto 1rem auto;
    }

    .loading-spinner.small {
      width: 30px;
      height: 30px;
      border-width: 3px;
    }

    @keyframes spin {
      0% { transform: rotate(0deg); }
      100% { transform: rotate(360deg); }
    }

    .error-state {
      color: var(--color-error);
    }

    .error-state h3 {
      color: var(--color-text);
      margin-bottom: 0.5rem;
    }

    .error-icon {
      font-size: 3rem;
      margin-bottom: 1rem;
    }

    .retry-button {
      background: var(--color-accent);
      color: #fff;
      border: none;
      padding: 0.6rem 1rem;
      border-radius: 8px;
      font-size: 0.9rem;
      font-weight: 600;
      cursor: pointer;
      margin-top: 1rem;
      transition: background 0.25s ease;
    }

    .retry-button:hover {
      opacity: 0.9;
    }

    .no-results-icon {
      font-size: 4rem;
      margin-bottom: 1rem;
    }

    .no-results h3 {
      color: var(--color-text);
      margin-bottom: 0.5rem;
    }

    .events-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
      gap: 1.5rem;
      margin-bottom: 2rem;
    }

    .event-card {
      background: var(--color-surface);
      border: 1px solid var(--color-border);
      border-radius: 12px;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      overflow: hidden;
      transition: all 0.2s ease;
      position: relative;
    }

    .event-card:hover {
      transform: translateY(-2px);
      box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
    }

    .event-card.highlighted {
      border-color: var(--color-accent);
      box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
    }

    .date-badge {
      position: absolute;
      top: 15px;
      right: 15px;
      background: var(--color-gradient);
      color: white;
      padding: 8px;
      border-radius: 8px;
      text-align: center;
      min-width: 60px;
      font-size: 0.85rem;
      font-weight: 600;
      z-index: 1;
    }

    .date-day {
      font-size: 1.2rem;
      font-weight: 700;
    }

    .date-month {
      font-size: 0.8rem;
      text-transform: uppercase;
      margin-top: 2px;
    }

    .event-content {
      padding: 20px;
      padding-right: 90px;
    }

    .event-header {
      margin-bottom: 1rem;
    }

    .event-title {
      font-size: 1.25rem;
      font-weight: 600;
      color: var(--color-text);
      margin: 0 0 0.5rem 0;
      line-height: 1.3;
    }

    .event-time {
      color: var(--color-text-muted);
      font-size: 0.9rem;
      font-weight: 500;
    }

    .venue-info {
      margin-bottom: 1rem;
    }

    .venue-name {
      font-weight: 600;
      color: var(--color-text);
      margin-bottom: 0.3rem;
    }

    .venue-address {
      color: var(--color-text-muted);
      font-size: 0.85rem;
    }

    .artists-section {
      margin-bottom: 1rem;
    }

    .artists-list {
      margin-bottom: 0.5rem;
    }

    .artist-name {
      font-weight: 600;
      color: var(--color-text);
    }

    .artist-name.highlighted-artist {
      background: var(--color-gradient);
      color: white;
      padding: 2px 6px;
      border-radius: 4px;
      font-weight: 700;
    }

    .genres {
      display: flex;
      flex-wrap: wrap;
      gap: 0.5rem;
    }

    .genre-tag {
      background: var(--color-border);
      color: var(--color-text-muted);
      padding: 4px 8px;
      border-radius: 12px;
      font-size: 0.8rem;
      font-weight: 500;
    }

    .ticket-section {
      margin-top: 1rem;
    }

    .ticket-button {
      display: inline-flex;
      align-items: center;
      gap: 0.4rem;
      background: var(--color-accent);
      color: white;
      padding: 0.6rem 1rem;
      border-radius: 8px;
      text-decoration: none;
      font-weight: 600;
      font-size: 0.9rem;
      transition: all 0.25s ease;
    }

    .ticket-button:hover {
      opacity: 0.9;
      transform: translateY(-1px);
    }

    .load-more-section {
      text-align: center;
      margin: 2rem 0;
    }

    .load-more-button {
      background: var(--color-accent);
      color: white;
      border: none;
      padding: 0.6rem 1rem;
      border-radius: 8px;
      cursor: pointer;
      font-size: 0.9rem;
      font-weight: 600;
      transition: all 0.25s ease;
    }

    .load-more-button:hover:not(:disabled) {
      opacity: 0.9;
    }

    .load-more-button:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }

    .loading-more {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 1rem;
      padding: 2rem;
      color: var(--color-text-muted);
    }

    @media (max-width: 768px) {
      .chicago-events-container {
        padding: 1rem 1rem 5rem;
      }
      
      .page-header h1 {
        font-size: 1.75rem;
      }
      
      .events-grid {
        grid-template-columns: 1fr;
        gap: 1rem;
      }
      
      .event-content {
        padding: 1rem;
        padding-right: 75px;
      }
      
      .date-badge {
        min-width: 50px;
        font-size: 0.8rem;
        top: 10px;
        right: 10px;
      }

      .search-section {
        padding: 1rem;
      }
    }
  `]
})
export class ChicagoEventsComponent implements OnInit, OnDestroy {
  events: ChicagoEvent[] = [];
  searchQuery = '';
  isLoading = false;
  isLoadingMore = false;
  hasMore = false;
  totalCount = 0;
  error: string | null = null;
  
  private destroy$ = new Subject<void>();
  private searchSubject = new BehaviorSubject<string>('');
  private currentOffset = 0;
  private readonly pageSize = 20;
  
  private apiBaseUrl: string;

  constructor(private http: HttpClient) {
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
}