import { Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { tap } from 'rxjs/operators';
import { environment } from '../environments/environment';

export interface User { email: string; fullName: string; artists?: string[]; }

const STORAGE_KEY = 'kmm_user';

@Injectable({ providedIn: 'root' })
export class AuthService {
  user = signal<User | null>(null);
  loading = signal(false);
  error = signal<string | null>(null);
  private apiBase = environment.apiBaseUrl;

  constructor(private http: HttpClient) {
    this.restore();
  }

  private restore(): void {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw) as User;
        if (parsed?.email && parsed?.fullName) {
          this.user.set(parsed);
        }
      }
    } catch {
      // ignore
    }
  }

  login(email: string, fullName: string) {
    this.loading.set(true);
    this.error.set(null);
    return this.http.post<User>(this.url('/login'), { email, fullName }).pipe(
      tap({
        next: (res) => {
          const u: User = { email, fullName, artists: res?.artists || undefined };
          this.user.set(u);
          localStorage.setItem(STORAGE_KEY, JSON.stringify(u));
          this.loading.set(false);
        },
        error: (err: any) => {
          this.loading.set(false);
          this.error.set(this.extractError(err));
        }
      })
    );
  }

  updateArtists(artists: string[]) {
    const u = this.user();
    if (u) {
      const updated = { ...u, artists };
      this.user.set(updated);
      localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
    }
  }

  logout(): void {
    this.user.set(null);
    localStorage.removeItem(STORAGE_KEY);
  }

  private url(path: string): string {
    return `${this.apiBase}${path}`;
  }

  private extractError(err: any): string {
    if (!err) return 'Unknown error';
    if (err.error) {
      if (typeof err.error === 'string') return err.error;
      if (typeof err.error.message === 'string') return err.error.message;
    }
    if (typeof err.message === 'string') return err.message;
    return 'Unexpected error';
  }
}
