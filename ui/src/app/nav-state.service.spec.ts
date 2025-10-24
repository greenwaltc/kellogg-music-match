import { TestBed } from '@angular/core/testing';
import { NavStateService } from './nav-state.service';

function withFreshStorage<T>(fn: () => T): T {
  const backup = new Map<string,string|null>();
  const keys = ['affyneVisitedMatches','affyneVisitedEvents'];
  for (const k of keys) backup.set(k, localStorage.getItem(k));
  try {
    keys.forEach(k => localStorage.removeItem(k));
    return fn();
  } finally {
    keys.forEach(k => {
      const v = backup.get(k);
      if (v === null) localStorage.removeItem(k); else localStorage.setItem(k, v as string);
    });
  }
}

describe('NavStateService', () => {
  it('persists visited flags', () => withFreshStorage(() => {
    const svc = TestBed.inject(NavStateService);
    expect(svc.visitedMatches()).toBeFalse();
    expect(svc.visitedEvents()).toBeFalse();
    svc.markMatchesVisited();
    svc.markEventsVisited();
    expect(localStorage.getItem('affyneVisitedMatches')).toBe('true');
    expect(localStorage.getItem('affyneVisitedEvents')).toBe('true');
  }));

  it('initializes from localStorage', () => withFreshStorage(() => {
    localStorage.setItem('affyneVisitedMatches','true');
    const svc = TestBed.inject(NavStateService);
    expect(svc.visitedMatches()).toBeTrue();
  }));
});
