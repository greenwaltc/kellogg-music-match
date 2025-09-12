import { Injectable, signal } from '@angular/core';

const THEME_KEY = 'kmm_theme';
export type ThemeMode = 'dark' | 'light';

@Injectable({ providedIn: 'root' })
export class ThemeService {
  mode = signal<ThemeMode>('dark');

  constructor() {
    const stored = localStorage.getItem(THEME_KEY) as ThemeMode | null;
    if (stored === 'light' || stored === 'dark') {
      this.mode.set(stored);
    } else {
      try { if (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) { this.mode.set('light'); } } catch {}
    }
    this.apply();
    if (window.matchMedia) {
      try {
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', e => {
          const storedNow = localStorage.getItem(THEME_KEY);
          if (!storedNow) { this.mode.set(e.matches ? 'dark' : 'light'); this.apply(); }
        });
      } catch {}
    }
  }

  toggle(): void {
    this.mode.set(this.mode() === 'dark' ? 'light' : 'dark');
    localStorage.setItem(THEME_KEY, this.mode());
    this.apply();
  }

  private apply(): void {
    const root = document.documentElement;
    root.classList.remove('theme-dark', 'theme-light');
    root.classList.add(this.mode() === 'dark' ? 'theme-dark' : 'theme-light');
  }
}
