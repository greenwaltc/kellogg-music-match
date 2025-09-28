import { Component, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { MatchService } from './match.service';
import { SimilarityMeterComponent } from './similarity-meter.component';

@Component({
  selector: 'app-matches',
  standalone: true,
  imports: [CommonModule, SimilarityMeterComponent],
  template: `
  <section>
    <h2>Your Top Music Matches</h2>
    <p class="note">Tip: check back here often! As more Kellogg students register, we build better matches.</p>
    
    <!-- Loading indicator -->
    <div *ngIf="matches.loading()" class="loading-container">
      <div class="loading-spinner"></div>
      <p class="loading-text">Finding your music matches...</p>
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
                <h4>Favorite Artists</h4>
                <div class="artists-grid">
                  <span 
                    *ngFor="let artist of match.artists" 
                    class="artist-tag"
                    [class.crush-artist]="match.name === 'Your Kellogg MBA Crush'">
                    {{ artist }}
                  </span>
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
  `,
  styles: [`
    .matches-list {
      display: flex;
      flex-direction: column;
      gap: 1.5rem;
      margin: 1.5rem 0;
    }

    .match-card {
      background: #fff;
      border: 1px solid #e0e0e0;
      border-radius: 12px;
      padding: 1.5rem;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      transition: transform 0.2s ease, box-shadow 0.2s ease;
    }

    .match-card:hover {
      transform: translateY(-2px);
      box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
    }

    .match-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 1rem;
      padding-bottom: 1rem;
      border-bottom: 1px solid #f0f0f0;
    }

    .match-name {
      margin: 0;
      font-size: 1.25rem;
      font-weight: 600;
      color: #2c3e50;
    }

    .match-details {
      display: grid;
      grid-template-columns: 2fr 1fr;
      gap: 1.5rem;
      align-items: start;
    }

    .artists-section h4 {
      margin: 0 0 0.75rem 0;
      font-size: 1rem;
      font-weight: 600;
      color: #34495e;
    }

    .artists-grid {
      display: flex;
      flex-wrap: wrap;
      gap: 0.5rem;
    }

    .artist-tag {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
      padding: 0.375rem 0.75rem;
      border-radius: 20px;
      font-size: 0.875rem;
      font-weight: 500;
      white-space: nowrap;
      transition: transform 0.2s ease;
    }

    .artist-tag:hover {
      transform: scale(1.05);
    }

    .match-stats {
      display: flex;
      flex-direction: column;
      gap: 0.75rem;
    }

    .stat {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .stat-label {
      font-size: 0.875rem;
      color: #7f8c8d;
      font-weight: 500;
    }

    .stat-value {
      font-size: 0.875rem;
      font-weight: 600;
      color: #2c3e50;
    }

    .empty-msg {
      text-align: center;
      color: #7f8c8d;
      font-style: italic;
      padding: 2rem;
      background: #f8f9fa;
      border-radius: 8px;
      margin: 1rem 0;
    }

    .note {
      text-align: center;
      color: #6c757d;
      font-size: 0.875rem;
      margin: 1.5rem 0;
      padding: 1rem;
      background: #e8f4f8;
      border-left: 4px solid #17a2b8;
      border-radius: 4px;
    }

    .loading-container {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 3rem 1rem;
      margin: 2rem 0;
    }

    .loading-spinner {
      width: 40px;
      height: 40px;
      border: 4px solid #f3f3f3;
      border-top: 4px solid #667eea;
      border-radius: 50%;
      animation: spin 1s linear infinite;
      margin-bottom: 1rem;
    }

    .loading-text {
      color: #6c757d;
      font-size: 1rem;
      font-weight: 500;
      margin: 0;
      text-align: center;
    }

    @keyframes spin {
      0% { transform: rotate(0deg); }
      100% { transform: rotate(360deg); }
    }

    .actions {
      text-align: center;
      margin-top: 2rem;
    }

    .actions button {
      background: #6c757d;
      color: white;
      border: none;
      padding: 0.75rem 2rem;
      border-radius: 6px;
      font-size: 1rem;
      cursor: pointer;
      transition: background-color 0.2s ease;
    }

    .match-card.crush-card {
      background: linear-gradient(135deg, #ffe6f0 0%, #fff0f5 100%);
      border: 2px solid #ff69b4;
      position: relative;
      overflow: hidden;
    }

    .match-card.crush-card::before {
      content: '💕';
      position: absolute;
      top: 10px;
      right: 15px;
      font-size: 1.5rem;
      opacity: 0.7;
    }

    .crush-message {
      background: #ffe6f0;
      border: 1px solid #ff69b4;
      border-radius: 8px;
      padding: 0.75rem;
      margin: 0.5rem 0 1rem 0;
      text-align: center;
    }

    .sorry-note {
      margin: 0;
      color: #d63384;
      font-weight: 500;
      font-style: italic;
    }

    .artist-tag.crush-artist {
      background: linear-gradient(135deg, #ff69b4 0%, #ff1493 100%);
      color: white;
    }

    .match-card.crush-card .match-name {
      color: #d63384;
      position: relative;
    }

    .match-card.crush-card .match-name::after {
      content: ' 💘';
      font-size: 0.8em;
    }

    @media (max-width: 768px) {
      .match-header {
        flex-direction: column;
        gap: 1rem;
        align-items: stretch;
      }

      .match-details {
        grid-template-columns: 1fr;
        gap: 1rem;
      }

      .artists-grid {
        max-height: 100px;
        overflow-y: auto;
      }
    }
  `]
})
export class MatchesComponent {
  constructor(public matches: MatchService, private router: Router) {
    effect(() => { if (!this.matches.matches()) { /* could redirect */ } });
    this.matches.fetchIfReady();
  }
  back(): void { this.router.navigateByUrl('/artists'); }
}
