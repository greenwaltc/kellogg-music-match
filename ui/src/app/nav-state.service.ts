import { Injectable, signal } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class NavStateService {
  private readonly kMatches = 'kmmVisitedMatches';
  private readonly kEvents = 'kmmVisitedEvents';

  visitedMatches = signal<boolean>(false);
  visitedEvents = signal<boolean>(false);

  constructor() {
    // Initialize from localStorage
    try {
      const m = localStorage.getItem(this.kMatches);
      const e = localStorage.getItem(this.kEvents);
      if (m === 'true') this.visitedMatches.set(true);
      if (e === 'true') this.visitedEvents.set(true);
    } catch {}
  }

  markMatchesVisited(): void {
    if (!this.visitedMatches()) {
      this.visitedMatches.set(true);
      try { localStorage.setItem(this.kMatches, 'true'); } catch {}
    }
  }

  markEventsVisited(): void {
    if (!this.visitedEvents()) {
      this.visitedEvents.set(true);
      try { localStorage.setItem(this.kEvents, 'true'); } catch {}
    }
  }

  /** When a user enters via Home tiles, reveal both primary menu items. */
  markAllPrimaryVisited(): void {
    this.markMatchesVisited();
    this.markEventsVisited();
  }

  /** Clear visited flags (used on login/register/logout to avoid leaking state across users). */
  resetVisited(): void {
    this.visitedMatches.set(false);
    this.visitedEvents.set(false);
    try {
      localStorage.removeItem(this.kMatches);
      localStorage.removeItem(this.kEvents);
    } catch {}
  }
}
