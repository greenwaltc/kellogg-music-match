import { Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { tap } from 'rxjs/operators';
import { environment } from '../environments/environment';

export interface User { 
  id?: string;
  username: string; 
  email: string; 
  firstName: string;
  lastName: string;
  artists?: string[]; 
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  email: string;
  firstName: string;
  lastName: string;
  password: string;
  program: string;
  graduationYear: number;
}

export interface AuthResponse {
  user: User;
  token?: string;
}

export interface ForgotPasswordRequest {
  email: string;
}

export interface ResetPasswordRequest {
  token: string;
  newPassword: string;
}

export interface MessageResponse {
  message: string;
}

const STORAGE_KEY = 'kmm_user';

@Injectable({ providedIn: 'root' })
export class AuthService {
  user = signal<User | null>(null);
  loading = signal(false);
  error = signal<string | null>(null);

  constructor(private http: HttpClient) {
    this.restore();
  }

  private restore(): void {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw) as User;
        if (parsed?.email && parsed?.firstName && parsed?.lastName) {
          this.user.set(parsed);
        }
      }
    } catch {
      // ignore
    }
  }

  login(loginData: LoginRequest) {
    this.loading.set(true);
    this.error.set(null);
    return this.http.post<AuthResponse>(this.url('/login'), loginData).pipe(
      tap({
        next: (res: AuthResponse) => {
          this.user.set(res.user);
          localStorage.setItem(STORAGE_KEY, JSON.stringify(res.user));
          if (res.token) {
            localStorage.setItem('kmm_token', res.token);
          }
          this.loading.set(false);
        },
        error: (err: any) => {
          this.loading.set(false);
          this.error.set(this.extractError(err));
        }
      })
    );
  }

  register(registerData: RegisterRequest) {
    this.loading.set(true);
    this.error.set(null);
    return this.http.post<AuthResponse>(this.url('/register'), registerData).pipe(
      tap({
        next: (res: AuthResponse) => {
          this.user.set(res.user);
          localStorage.setItem(STORAGE_KEY, JSON.stringify(res.user));
          if (res.token) {
            localStorage.setItem('kmm_token', res.token);
          }
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
    localStorage.removeItem('kmm_token');
  }

  forgotPassword(email: string) {
    this.loading.set(true);
    this.error.set(null);
    return this.http.post<MessageResponse>(this.url('/forgot-password'), { email }).pipe(
      tap({
        next: () => {
          this.loading.set(false);
        },
        error: (err: any) => {
          this.loading.set(false);
          this.error.set(this.extractError(err));
        }
      })
    );
  }

  resetPassword(token: string, newPassword: string) {
    this.loading.set(true);
    this.error.set(null);
    return this.http.post<MessageResponse>(this.url('/reset-password'), { token, newPassword }).pipe(
      tap({
        next: () => {
          this.loading.set(false);
        },
        error: (err: any) => {
          this.loading.set(false);
          this.error.set(this.extractError(err));
        }
      })
    );
  }

  private url(path: string): string {
    const apiUrl = window.__kmmConfig?.apiBaseUrl || environment.apiBaseUrl;
    return `${apiUrl}${path}`;
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
