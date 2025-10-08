import { Component, EventEmitter, Output, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { SpotifyService } from './spotify.service';

@Component({
  selector: 'app-spotify-connect',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="spotify-connect-bar">
      <button type="button" class="spotify-btn" (click)="connect()" [disabled]="loading">
        <ng-container *ngIf="!loading">🎧 {{ label }}</ng-container>
        <ng-container *ngIf="loading">Connecting…</ng-container>
      </button>
    </div>
  `
})
export class SpotifyConnectComponent {
  @Input() label = 'Connect / Sync Spotify';
  @Output() connected = new EventEmitter<void>();
  loading = false;
  constructor(private spotify: SpotifyService) {}
  async connect() {
    if (this.loading) return;
    this.loading = true;
    try {
      await this.spotify.beginAuth();
      this.connected.emit();
    } finally {
      this.loading = false;
    }
  }
}