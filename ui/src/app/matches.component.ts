import { Component, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { MatchService } from './match.service';

@Component({
  selector: 'app-matches',
  standalone: true,
  imports: [CommonModule],
  template: `
  <section>
    <h2>Your Top Music Matches</h2>
    <ng-container *ngIf="matches.matches() as list; else noFetchYet">
      <ng-container *ngIf="list.length; else noMatchesYet">
        <ol>
          <li *ngFor="let m of list">{{ m.name || (m | json) }}</li>
        </ol>
      </ng-container>
      <ng-template #noMatchesYet>
        <p class="empty-msg">No matches yet, but come back soon! We need more users to generate high-quality matches.</p>
      </ng-template>
    </ng-container>
    <ng-template #noFetchYet>
      <p class="empty-msg">No matches retrieved yet. Add or update your favorite artists to get started.</p>
    </ng-template>
    <p class="note">Tip: check back here often! As more Kellogg students register, we build better matches.</p>
    <div class="actions">
      <button type="button" (click)="back()">Back</button>
    </div>
  </section>
  `
})
export class MatchesComponent {
  constructor(public matches: MatchService, private router: Router) {
    effect(() => { if (!this.matches.matches()) { /* could redirect */ } });
  }
  back(): void { this.router.navigateByUrl('/artists'); }
}
