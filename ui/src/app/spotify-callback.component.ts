import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { SpotifyService } from './spotify.service';

@Component({
  selector: 'app-spotify-callback',
  standalone: true,
  imports: [CommonModule],
  template: `
    <section class="spotify-callback">
      <h2>Spotify Authorization</h2>
      <ng-container *ngIf="phase === 'processing'; else doneTpl">
        <p>Processing authorization…</p>
      </ng-container>
      <ng-template #doneTpl>
        <p *ngIf="error" class="error">{{ error }}</p>
        <p *ngIf="!error">You may close this page. Sync will continue in the background.</p>
      </ng-template>
    </section>
  `,
  styles: [`
    .spotify-callback { max-width: 600px; margin: 2rem auto; }
    .error { color: #c00; }
  `]
})
export class SpotifyCallbackComponent implements OnInit {
  phase: 'processing' | 'done' = 'processing';
  error: string | null = null;
  constructor(private route: ActivatedRoute, private router: Router, private spotify: SpotifyService) {}
  ngOnInit(): void {
    const code = this.route.snapshot.queryParamMap.get('code');
    const state = this.route.snapshot.queryParamMap.get('state') || '';
    if (!code) { this.error = 'Missing authorization code'; this.phase = 'done'; return; }
    // For now, we call backend, swallow any errors, and finish.
    this.spotify.exchangeCode(code, state).subscribe({
      next: () => { this.phase = 'done'; },
      error: (err: any) => { this.error = err?.error?.message || err.message || 'Exchange failed'; this.phase = 'done'; }
    });
  }
}