import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { SpotifyService } from './spotify.service';

@Component({
  selector: 'app-spotify-connect',
  standalone: true,
  imports: [CommonModule],
  template: `
    <section class="spotify-connect">
      <h2>Connect Your Spotify Account</h2>
      <p>Authorize the app to read your top artists and playlists to improve matching.</p>
      <button type="button" (click)="connect()" [disabled]="busy">Connect with Spotify</button>
      <p *ngIf="error" class="error">{{ error }}</p>
    </section>
  `,
  styles: [`
    .spotify-connect { max-width: 600px; margin: 2rem auto; }
    button { padding: 0.6rem 1rem; font-size: 1rem; }
    .error { color: #c00; }
  `]
})
export class SpotifyConnectComponent {
  busy = false;
  error: string | null = null;
  constructor(private spotify: SpotifyService) {}
  async connect() {
    this.busy = true; this.error = null;
    try {
      await this.spotify.beginAuth();
    } catch (e: any) {
      this.error = e?.message || 'Failed to initiate Spotify auth';
      this.busy = false;
    }
  }
}