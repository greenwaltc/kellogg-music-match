import { Component, signal, effect, computed, ElementRef, ViewChild, AfterViewInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { EventsService, EventsSearchFilters, EventDTO, EventsPage } from './events.service';
import { debounceTime, Subject, switchMap, catchError, of } from 'rxjs';

@Component({
  selector: 'app-events-search',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
  <div class="events-search-container">
    <div class="page-header">
      <h1>🎫 Find Events</h1>
      <p class="subtitle">Search Ticketmaster across any location and type</p>
    </div>

    <form (ngSubmit)="runSearch(true)" class="filters-form" novalidate>
      <div class="row">
        <label>
          <span>Keyword</span>
          <input type="text" [(ngModel)]="filters.keyword" name="keyword" (keyup.enter)="runSearch(true)" autocomplete="off"/>
        </label>
        <label>
          <span>Segment</span>
          <input type="text" [(ngModel)]="filters.segmentName" name="segmentName" placeholder="Music, Sports, Arts & Theatre..." />
        </label>
        <label>
          <span>Classification</span>
          <input type="text" [(ngModel)]="filters.classificationName" name="classificationName" placeholder="Rock, Pop, Comedy..."/>
        </label>
      </div>

      <div class="row">
        <label>
          <span>City</span>
          <input type="text" [(ngModel)]="filters.city" name="city" placeholder="Chicago"/>
        </label>
        <label>
          <span>State</span>
          <input type="text" [(ngModel)]="filters.stateCode" name="stateCode" placeholder="IL"/>
        </label>
        <label>
          <span>Country</span>
          <input type="text" [(ngModel)]="filters.countryCode" name="countryCode" placeholder="US"/>
        </label>
      </div>

      <div class="row">
        <label>
          <span>Lat,Long</span>
          <input type="text" [(ngModel)]="filters.latlong" name="latlong" placeholder="41.8781,-87.6298"/>
        </label>
        <label>
          <span>Radius (mi)</span>
          <input type="number" [(ngModel)]="filters.radius" name="radius" min="1" max="200" />
        </label>
        <label>
          <span>Sort</span>
          <select [(ngModel)]="filters.sort" name="sort">
            <option value="">Default</option>
            <option value="date,asc">Date ↑</option>
            <option value="date,desc">Date ↓</option>
            <option value="name,asc">Name ↑</option>
            <option value="name,desc">Name ↓</option>
          </select>
        </label>
      </div>

      <div class="row">
        <label>
          <span>Start</span>
          <input type="datetime-local" [(ngModel)]="startLocal" name="startLocal"/>
        </label>
        <label>
          <span>End</span>
          <input type="datetime-local" [(ngModel)]="endLocal" name="endLocal"/>
        </label>
        <label class="checkbox">
          <input type="checkbox" [(ngModel)]="includeAssociated" name="includeAssociated"/>
          <span>Include my associations</span>
        </label>
      </div>

      <div class="actions">
        <button type="submit" [disabled]="loading">Search</button>
        <button type="button" (click)="clear()" [disabled]="loading">Clear</button>
      </div>
    </form>

    <div class="hint" *ngIf="hint">{{hint}}</div>
    <div class="error" *ngIf="error">{{error}}</div>

    <div class="results" *ngIf="items.length">
      <div class="summary">Showing {{items.length}} of {{total ?? '—'}} results</div>
      <div class="grid">
        <div class="card" *ngFor="let ev of items; trackBy: trackId">
          <div class="title">{{ev.name}}</div>
          <div class="meta">{{ev.city || ''}} {{ev.state || ''}} {{ev.country || ''}}</div>
          <div class="time" *ngIf="ev.startUtc">{{ev.startUtc | date:'medium'}}</div>
          <a class="link" *ngIf="ev.url" [href]="ev.url" target="_blank" rel="noopener">Open</a>
          <div class="assoc" *ngIf="ev.association">
            <span>Mine: {{ev.association?.myStatus || 'NONE'}}</span>
            <span>Interested: {{ev.association?.interestedCount || 0}}</span>
            <span>Going: {{ev.association?.goingCount || 0}}</span>
            <span>LFG: {{ev.association?.lfgCount || 0}}</span>
          </div>
          <div class="actions" *ngIf="ev.externalId || ev.id">
            <button (click)="mark(ev,'INTERESTED')" [disabled]="loading">★ Interested</button>
            <button (click)="mark(ev,'GOING')" [disabled]="loading">✅ Going</button>
            <button (click)="mark(ev,'LFG')" [disabled]="loading">👥 LFG</button>
            <button (click)="clearAssociation(ev)" [disabled]="loading">✖ Clear</button>
          </div>
        </div>
      </div>
      <div #sentinel class="sentinel" *ngIf="items.length"></div>
      <div class="pager">
        <button (click)="prevPage()" [disabled]="page===0 || loading">Prev</button>
        <span>Page {{page+1}}</span>
        <button (click)="nextPage()" [disabled]="loading || (total !== undefined && (page+1)*size >= (total||0))">Next</button>
      </div>
    </div>
  </div>
  `,
  styles: [`
    .events-search-container{max-width:980px;margin:1rem auto;padding:0 1rem}
    .page-header h1{margin:.2rem 0}
    form.filters-form{display:flex;flex-direction:column;gap:.75rem;background:var(--color-bg-soft);padding:1rem;border-radius:12px;border:1px solid var(--color-border)}
    .row{display:flex;gap:.75rem;flex-wrap:wrap}
    label{display:flex;flex-direction:column;gap:.35rem;min-width:180px;flex:1}
    input,select{padding:.5rem .6rem;border-radius:8px;border:1px solid var(--color-border);background:var(--color-bg);color:var(--color-text)}
    .checkbox{flex-direction:row;align-items:center;gap:.5rem}
    .actions{display:flex;gap:.5rem}
    .results .grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:.75rem;margin-top:.75rem}
    .card{border:1px solid var(--color-border);border-radius:12px;padding:.75rem;background:var(--color-bg)}
    .title{font-weight:600}
    .summary{margin-top:1rem}
    .pager{display:flex;gap:1rem;align-items:center;margin-top:.75rem}
    .hint{color:var(--color-text-muted);margin-top:.5rem}
    .error{color:#b00020;margin-top:.5rem}
  `]
})
export class EventsSearchComponent implements AfterViewInit {
  filters: EventsSearchFilters = { includeAssociated: true };
  startLocal = '';
  endLocal = '';
  includeAssociated = true;

  page = 0;
  size = 20;
  total?: number;
  items: EventDTO[] = [];
  loading = false;
  error: string | null = null;
  hint: string | null = 'Tip: start with a keyword or set a city/radius';

  private search$ = new Subject<void>();
  private appendMode = false;
  private lastBatchLen = 0;
  @ViewChild('sentinel') sentinel?: ElementRef<HTMLDivElement>;
  private io?: IntersectionObserver;

  constructor(private events: EventsService) {
    this.search$
      .pipe(
        debounceTime(150),
        switchMap(() => {
          this.loading = true; this.error = null;
          // map date-local to ISO if present
          const f: EventsSearchFilters = { ...this.filters };
          f.includeAssociated = this.includeAssociated;
          f.startDateTime = this.startLocal ? new Date(this.startLocal).toISOString() : undefined;
          f.endDateTime = this.endLocal ? new Date(this.endLocal).toISOString() : undefined;
          return this.events.search(f, this.page, this.size).pipe(
            catchError((err) => {
              this.error = err?.error?.message || 'Search failed';
              return of({ page: this.page, size: this.size, items: [], total: 0 } as EventsPage);
            })
          );
        })
      )
      .subscribe((res) => {
        const batch = res.items || [];
        if (this.appendMode) {
          this.items = [...this.items, ...batch];
        } else {
          this.items = batch;
        }
        this.lastBatchLen = batch.length;
        this.appendMode = false;
        this.total = res.total;
        this.loading = false;
      });
  }

  runSearch(reset = false) {
    if (reset) this.page = 0;
    this.hint = null;
    this.search$.next();
  }

  clear() {
    this.filters = { includeAssociated: true };
    this.startLocal = this.endLocal = '';
    this.includeAssociated = true;
    this.page = 0; this.total = undefined; this.items = [];
    this.hint = 'Tip: start with a keyword or set a city/radius';
    this.error = null;
  }

  prevPage() { if (this.page > 0) { this.page--; this.runSearch(); } }
  nextPage() { this.page++; this.runSearch(); }

  ngAfterViewInit(): void {
    if (typeof window === 'undefined' || !('IntersectionObserver' in window)) return;
    this.io = new IntersectionObserver(entries => {
      const entry = entries[0];
      if (entry.isIntersecting && this.canLoadMore()) {
        this.appendMode = true;
        this.page++;
        this.runSearch();
      }
    }, { root: null, rootMargin: '0px', threshold: 0.1 });
    setTimeout(() => {
      const el = this.sentinel?.nativeElement;
      if (el) this.io?.observe(el);
    });
  }

  private canLoadMore(): boolean {
    if (this.loading) return false;
    if (this.total !== undefined) return (this.page + 1) * this.size < (this.total || 0);
    return this.lastBatchLen === this.size; // heuristic when total unknown
  }

  trackId(i: number, ev: EventDTO) { return ev.id || ev.externalId || i; }

  private eventKey(ev: EventDTO) {
    return ev.id || ev.externalId || '';
  }

  mark(ev: EventDTO, status: 'INTERESTED'|'GOING'|'LFG') {
    const id = this.eventKey(ev);
    if (!id) return;
    // optimistic update
    ev.association = ev.association || { myStatus: 'NONE', interestedCount: 0, goingCount: 0, lfgCount: 0 };
    ev.association.myStatus = status as any;
    this.events.setAssociation(id, status).subscribe({
      next: () => {},
      error: () => { this.error = 'Failed to update association'; }
    });
  }

  clearAssociation(ev: EventDTO) {
    const id = this.eventKey(ev);
    if (!id) return;
    this.events.clearAssociation(id).subscribe({
      next: () => { if (ev.association) ev.association.myStatus = 'NONE'; },
      error: () => { this.error = 'Failed to clear association'; }
    });
  }
}
