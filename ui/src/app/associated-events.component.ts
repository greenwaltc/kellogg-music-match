import { Component, AfterViewInit, ElementRef, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { EventsService, EventDTO, EventsPage } from './events.service';
import { catchError, debounceTime, of, Subject, switchMap } from 'rxjs';

@Component({
  selector: 'app-associated-events',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
  <div class="events-associated-container">
    <div class="page-header">
      <h1>⭐ My Associated Events</h1>
      <p class="subtitle">All events you or classmates have marked (no Ticketmaster call)</p>
    </div>

    <form (ngSubmit)="runSearch(true)" class="filters-form" novalidate>
      <div class="row">
        <label>
          <span>City</span>
          <input type="text" [(ngModel)]="city" name="city" placeholder="City" />
        </label>
        <label>
          <span>Segment</span>
          <input type="text" [(ngModel)]="segmentName" name="segmentName" placeholder="Music, Sports..." />
        </label>
        <label>
          <span>Start</span>
          <input type="datetime-local" [(ngModel)]="startLocal" name="startLocal" />
        </label>
        <label>
          <span>End</span>
          <input type="datetime-local" [(ngModel)]="endLocal" name="endLocal" />
        </label>
      </div>
      <div class="actions">
        <button type="submit" [disabled]="loading">Apply</button>
        <button type="button" (click)="clear()" [disabled]="loading">Clear</button>
      </div>
    </form>

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
            <span>Mine: {{ev.association.myStatus || 'NONE'}}</span>
            <span>Interested: {{ev.association.interestedCount || 0}}</span>
            <span>Going: {{ev.association.goingCount || 0}}</span>
            <span>LFG: {{ev.association.lfgCount || 0}}</span>
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
    .events-associated-container{max-width:980px;margin:1rem auto;padding:0 1rem}
    .page-header h1{margin:.2rem 0}
    form.filters-form{display:flex;flex-direction:column;gap:.75rem;background:var(--color-bg-soft);padding:1rem;border-radius:12px;border:1px solid var(--color-border)}
    .row{display:flex;gap:.75rem;flex-wrap:wrap}
    label{display:flex;flex-direction:column;gap:.35rem;min-width:180px;flex:1}
    input,select{padding:.5rem .6rem;border-radius:8px;border:1px solid var(--color-border);background:var(--color-bg);color:var(--color-text)}
    .actions{display:flex;gap:.5rem}
    .results .grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:.75rem;margin-top:.75rem}
    .card{border:1px solid var(--color-border);border-radius:12px;padding:.75rem;background:var(--color-bg)}
    .title{font-weight:600}
    .summary{margin-top:1rem}
    .pager{display:flex;gap:1rem;align-items:center;margin-top:.75rem}
    .error{color:#b00020;margin-top:.5rem}
  `]
})
export class AssociatedEventsComponent implements AfterViewInit {
  // Filters
  city = '';
  segmentName = '';
  startLocal = '';
  endLocal = '';

  // Paging/results
  page = 0;
  size = 20;
  total?: number;
  items: EventDTO[] = [];
  loading = false;
  error: string | null = null;

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
          const params: any = {};
          if (this.city) params.city = this.city;
          if (this.segmentName) params.segmentName = this.segmentName;
          params.startDateTime = this.startLocal ? new Date(this.startLocal).toISOString() : undefined;
          params.endDateTime = this.endLocal ? new Date(this.endLocal).toISOString() : undefined;
          return this.events.getAssociated(params, this.page, this.size).pipe(
            catchError((err) => { this.error = err?.error?.message || 'Fetch failed'; return of({ page: this.page, size: this.size, items: [], total: 0 } as EventsPage); })
          );
        })
      )
      .subscribe(res => {
        const batch = res.items || [];
        if (this.appendMode) { this.items = [...this.items, ...batch]; } else { this.items = batch; }
        this.lastBatchLen = batch.length;
        this.appendMode = false;
        this.total = res.total;
        this.loading = false;
      });

    // Initial load
    queueMicrotask(() => this.runSearch(true));
  }

  runSearch(reset = false) { if (reset) this.page = 0; this.search$.next(); }
  clear() { this.city=''; this.segmentName=''; this.startLocal=''; this.endLocal=''; this.page=0; this.total=undefined; this.items=[]; this.error=null; }
  prevPage() { if (this.page>0) { this.page--; this.runSearch(); } }
  nextPage() { this.page++; this.runSearch(); }

  ngAfterViewInit(): void {
    if (typeof window === 'undefined' || !('IntersectionObserver' in window)) return;
    this.io = new IntersectionObserver(entries => {
      const entry = entries[0];
      if (entry.isIntersecting && this.canLoadMore()) {
        this.appendMode = true; this.page++; this.runSearch();
      }
    }, { root: null, rootMargin: '0px', threshold: 0.1 });
    setTimeout(() => { const el = this.sentinel?.nativeElement; if (el) this.io?.observe(el); });
  }

  private canLoadMore(): boolean {
    if (this.loading) return false;
    if (this.total !== undefined) return (this.page + 1) * this.size < (this.total || 0);
    return this.lastBatchLen === this.size; // heuristic
  }

  trackId(i: number, ev: EventDTO) { return ev.id || ev.externalId || i; }

  private eventKey(ev: EventDTO) { return ev.id || ev.externalId || ''; }

  mark(ev: EventDTO, status: 'INTERESTED'|'GOING'|'LFG') {
    const id = this.eventKey(ev); if (!id) return;
    ev.association = ev.association || { myStatus: 'NONE', interestedCount: 0, goingCount: 0, lfgCount: 0 };
    ev.association.myStatus = status as any;
    this.events.setAssociation(id, status).subscribe({ error: () => { this.error = 'Failed to update association'; } });
  }

  clearAssociation(ev: EventDTO) {
    const id = this.eventKey(ev); if (!id) return;
    this.events.clearAssociation(id).subscribe({ next: () => { if (ev.association) ev.association.myStatus = 'NONE'; }, error: () => { this.error = 'Failed to clear association'; } });
  }
}
