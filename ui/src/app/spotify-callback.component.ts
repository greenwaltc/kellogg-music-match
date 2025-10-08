import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { SpotifyService } from './spotify.service';
import { HttpClient } from '@angular/common/http';
import { ApiBaseService } from './api-base.service';
import { MatchService } from './match.service';

@Component({
  selector: 'app-spotify-callback',
  standalone: true,
  imports: [CommonModule],
  template: `
    <section class="spotify-callback">
      <h2>Spotify Authorization</h2>
      <ng-container *ngIf="phase === 'processing'; else doneTpl">
        <div class="spinner" aria-label="Loading" role="status"></div>
        <p *ngIf="!syncing">Processing authorization…</p>
        <ng-container *ngIf="syncing">
          <p *ngIf="status !== 'failed' && status !== 'cancelled'">
            Syncing your Spotify data<span *ngIf="progress !== null">… {{ progress }}%</span>
          </p>
          <p *ngIf="status === 'failed'" class="error">Sync failed. {{ statusMessage || 'Please retry.' }} <button type="button" (click)="retry()">Retry</button></p>
          <p *ngIf="status === 'cancelled'" class="warn">Sync cancelled. <button type="button" (click)="retry()">Retry</button></p>
          <div class="actions" *ngIf="status !== 'failed' && status !== 'cancelled'">
            <button type="button" (click)="cancel()" [disabled]="cancelling">Cancel</button>
          </div>
        </ng-container>
      </ng-container>
      <ng-template #doneTpl>
        <p *ngIf="denied && !error" class="warn">Spotify access is required to find musically similar students and concerts relevant to you. However, you can still interact with the concerts page!</p>
        <button *ngIf="denied && !error" (click)="goConcerts()">Go to Chicago Concerts</button>
        <p *ngIf="error" class="error">{{ error }}</p>
        <p *ngIf="!error && !denied">Done.</p>
      </ng-template>
    </section>
  `,
  styles: [`
    .spotify-callback { max-width: 600px; margin: 2rem auto; }
    .error { color: #c00; }
    .warn { color: #8a6d00; }
    .spinner { width:48px; height:48px; border:5px solid #ccc; border-top-color:#1db954; border-radius:50%; animation: spin 1s linear infinite; margin:1rem auto; }
    @keyframes spin { to { transform: rotate(360deg); } }
  `]
})
export class SpotifyCallbackComponent implements OnInit {
  phase: 'processing' | 'done' = 'processing';
  error: string | null = null;
  denied = false;
  syncing = false;
  progress: number | null = null;
  status: string | null = null;
  statusMessage: string | null = null;
  cancelling = false;
  origCode: string | null = null;
  origState: string | null = null;
  constructor(private route: ActivatedRoute, private router: Router, private spotify: SpotifyService, private http: HttpClient, private api: ApiBaseService, private matches: MatchService) {}
  ngOnInit(): void {
    const code = this.route.snapshot.queryParamMap.get('code');
    const state = this.route.snapshot.queryParamMap.get('state') || '';
  this.origCode = code;
  this.origState = state;
    // If user denied (Spotify sends error=access_denied), show message & button.
    const errorParam = this.route.snapshot.queryParamMap.get('error');
    if (errorParam) {
      this.denied = true;
      this.phase = 'done';
      return;
    }
    if (!code) { this.error = 'Missing authorization code'; this.phase = 'done'; return; }
    // Begin token exchange then start sync spinner
    this.spotify.exchangeCode(code, state).subscribe({
      next: () => {
        this.error = null;
        this.phase = 'processing';
        this.syncing = true;
        // Start polling status directly (sync already started)
        this.pollStatus(0);
      },
      error: (err: any) => { this.error = err?.error?.message || err.message || 'Exchange failed'; this.phase = 'done'; }
    });
  }

  private pollStatus(attempt: number) {
    if (attempt > 20) { // timeout ~20 * 1s
      this.error = 'Sync timeout';
      this.phase = 'done';
      this.syncing = false;
      return;
    }
  this.http.get<{status:string; progress?: number; message?: string}>(this.api.url('/sync/spotify/status')).subscribe({
      next: (res) => {
        this.status = res.status;
        if (typeof res.progress === 'number') { this.progress = res.progress; }
        if (res.message) { this.statusMessage = res.message; }
        if (res.status === 'complete') {
          this.phase = 'done';
          this.syncing = false;
          // Refresh matches now that Spotify data is persisted
          this.matches.clear();
          this.matches.fetch('medium_term', 10);
          this.router.navigate(['/matches']);
        } else if (res.status === 'failed' || res.status === 'cancelled') {
          this.phase = 'done';
          this.syncing = false;
        } else {
          setTimeout(() => this.pollStatus(attempt + 1), 1000);
        }
      },
      error: () => { this.error = 'Sync failed'; setTimeout(() => this.pollStatus(attempt + 1), 1000); }
    });
  }

  cancel() {
    this.cancelling = true;
  this.http.delete(this.api.url('/sync/spotify')).subscribe({
      next: () => { this.status = 'cancelled'; this.phase = 'done'; this.syncing = false; },
      error: () => { this.cancelling = false; }
    });
  }
  
  retry() {
    if (!this.origCode || !this.origState) { return; }
    this.error = null;
    this.statusMessage = null;
    this.phase = 'processing';
    this.syncing = true;
    this.progress = 0;
  this.http.post(this.api.url('/sync/spotify/retry'), { code: this.origCode, state: this.origState }).subscribe({
      next: () => this.pollStatus(0),
      error: () => { this.error = 'Retry failed'; this.phase = 'done'; this.syncing = false; }
    });
  }

  goConcerts() { this.router.navigate(['/concerts']); }
}