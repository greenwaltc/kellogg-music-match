import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormArray, FormBuilder, FormControl, Validators, FormGroup } from '@angular/forms';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Router } from '@angular/router';
import { MatchService, MatchUser } from './match.service';
import { environment } from '../environments/environment';
import { AuthService } from './auth.service';

interface ArtistsFormShape { artists: FormArray<FormControl<string | null>>; }

@Component({
  selector: 'app-artists',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  template: `
  <section class="artists-section">
    <div class="page-header">
      <h2>Your Favorite Artists</h2>
      <p class="subtitle">List from most to least favorite. Add up to {{ maxArtists }} artists.</p>
    </div>
    
    <form [formGroup]="form" (ngSubmit)="submit()" novalidate class="artists-form">
      <div class="form-card">
        <div class="form-header">
          <h3>Top {{ artistsArray.length }} Artist{{ artistsArray.length !== 1 ? 's' : '' }}</h3>
          <span class="count-badge">{{ artistsArray.length }}/{{ maxArtists }}</span>
        </div>
        
        <div formArrayName="artists" class="artists-list">
          <div class="artist-row" *ngFor="let ctrl of artistsArray.controls; let i = index">
            <div class="input-wrapper">
              <span class="rank-number">{{ i + 1 }}</span>
              <input 
                [formControlName]="i" 
                type="text" 
                [attr.placeholder]="'Enter artist #' + (i+1)" 
                class="artist-input"
                [class.error]="ctrl.invalid && ctrl.touched"
              />
              <button 
                type="button" 
                class="remove-btn" 
                (click)="remove(i)" 
                *ngIf="artistsArray.length > 1" 
                aria-label="Remove artist"
                title="Remove this artist">
                ×
              </button>
            </div>
            <div class="error-msg" *ngIf="ctrl.invalid && ctrl.touched">
              <span *ngIf="ctrl.errors?.['required']">Artist name is required</span>
              <span *ngIf="ctrl.errors?.['minlength']">Artist name must be at least 2 characters</span>
            </div>
          </div>
        </div>
        
        <div class="form-actions">
          <button 
            type="button" 
            class="add-btn" 
            (click)="add()" 
            [disabled]="artistsArray.length >= maxArtists">
            <span class="btn-icon">＋</span>
            Add Artist
          </button>
          <span *ngIf="artistsArray.length >= maxArtists" class="limit-msg">
            Maximum of {{ maxArtists }} artists reached
          </span>
        </div>
      </div>
      
      <div class="submit-section">
        <button 
          type="submit" 
          class="submit-btn" 
          [disabled]="loading() || form.invalid">
          <span *ngIf="!loading()" class="btn-content">
            <span class="btn-icon">🎵</span>
            Find My Music Matches
          </span>
          <span *ngIf="loading()" class="btn-content loading">
            <span class="spinner"></span>
            Finding Matches...
          </span>
        </button>
        
        <div class="error-alert" *ngIf="error()">
          <span class="error-icon">⚠️</span>
          {{ error() }}
        </div>
      </div>
    </form>
  </section>
  `,
  styles: [`
    .artists-section {
      max-width: 800px;
      margin: 0 auto;
      padding: 2rem 1rem;
    }

    .page-header {
      text-align: center;
      margin-bottom: 2rem;
    }

    .page-header h2 {
      margin: 0 0 0.5rem 0;
      font-size: 2rem;
      font-weight: 700;
      color: #2c3e50;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
    }

    .subtitle {
      margin: 0;
      color: #6c757d;
      font-size: 1rem;
    }

    .form-card {
      background: #fff;
      border: 1px solid #e0e0e0;
      border-radius: 12px;
      padding: 2rem;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      margin-bottom: 2rem;
    }

    .form-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 1.5rem;
      padding-bottom: 1rem;
      border-bottom: 1px solid #f0f0f0;
    }

    .form-header h3 {
      margin: 0;
      font-size: 1.25rem;
      font-weight: 600;
      color: #2c3e50;
    }

    .count-badge {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
      padding: 0.25rem 0.75rem;
      border-radius: 20px;
      font-size: 0.875rem;
      font-weight: 600;
    }

    .artists-list {
      display: flex;
      flex-direction: column;
      gap: 1rem;
    }

    .input-wrapper {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      background: #f8f9fa;
      border: 2px solid #e9ecef;
      border-radius: 8px;
      padding: 0.75rem;
      transition: all 0.2s ease;
    }

    .input-wrapper:focus-within {
      background: #fff;
      border-color: #667eea;
      box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
    }

    .rank-number {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 2rem;
      height: 2rem;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
      border-radius: 50%;
      font-weight: 600;
      font-size: 0.875rem;
    }

    .artist-input {
      flex: 1;
      border: none;
      background: transparent;
      font-size: 1rem;
      color: #2c3e50;
      outline: none;
    }

    .remove-btn {
      width: 2rem;
      height: 2rem;
      background: #dc3545;
      color: white;
      border: none;
      border-radius: 50%;
      font-size: 1.25rem;
      cursor: pointer;
      transition: transform 0.2s ease;
    }

    .remove-btn:hover {
      transform: scale(1.1);
    }

    .error-msg {
      padding-left: 2.75rem;
      color: #dc3545;
      font-size: 0.875rem;
    }

    .form-actions {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-top: 1.5rem;
      padding-top: 1rem;
      border-top: 1px solid #f0f0f0;
    }

    .add-btn {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      background: linear-gradient(135deg, #28a745 0%, #20c997 100%);
      color: white;
      border: none;
      padding: 0.75rem 1.5rem;
      border-radius: 8px;
      font-weight: 600;
      cursor: pointer;
      transition: transform 0.2s ease;
    }

    .add-btn:hover:not(:disabled) {
      transform: translateY(-2px);
    }

    .add-btn:disabled {
      background: #6c757d;
      opacity: 0.6;
    }

    .submit-btn {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
      border: none;
      padding: 1rem 2rem;
      border-radius: 12px;
      font-size: 1.125rem;
      font-weight: 600;
      cursor: pointer;
      transition: transform 0.3s ease;
      min-width: 250px;
      margin: 0 auto;
      display: block;
    }

    .submit-btn:hover:not(:disabled) {
      transform: translateY(-3px);
    }

    .submit-btn:disabled {
      background: #6c757d;
      transform: none;
    }

    .btn-content {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 0.5rem;
    }

    .spinner {
      width: 1rem;
      height: 1rem;
      border: 2px solid transparent;
      border-top: 2px solid currentColor;
      border-radius: 50%;
      animation: spin 1s linear infinite;
    }

    @keyframes spin {
      to { transform: rotate(360deg); }
    }

    .error-alert {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      background: #f8d7da;
      color: #721c24;
      padding: 1rem;
      border-radius: 8px;
      margin-top: 1rem;
      justify-content: center;
    }

    @media (max-width: 768px) {
      .artists-section { padding: 1rem; }
      .form-card { padding: 1.5rem; }
      .form-actions { flex-direction: column; gap: 1rem; }
      .submit-btn { width: 100%; min-width: auto; }
    }
  `]
})
export class ArtistsComponent {
  maxArtists = 10;
  loading = signal(false);
  error = signal<string | null>(null);
  form!: FormGroup<ArtistsFormShape>;
  private apiBase = environment.apiBaseUrl;

  constructor(private fb: FormBuilder, private http: HttpClient, private matches: MatchService, private router: Router, private auth: AuthService) {
    this.form = this.fb.group<ArtistsFormShape>({ artists: this.fb.array([this.artistControl()]) });
    // Prefill for returning users
    const user = this.auth.user();
    if (user?.artists?.length) {
      const arr = this.artistsArray;
      while (arr.length) arr.removeAt(0);
      user.artists.slice(0, this.maxArtists).forEach((a: string) => arr.push(this.fb.control<string | null>(a, [Validators.required, Validators.minLength(2)])));
    }
  }

  get artistsArray(): FormArray<FormControl<string | null>> { return this.form.controls.artists; }
  private artistControl(): FormControl<string | null> { return this.fb.control<string | null>('', [Validators.required, Validators.minLength(2)]); }

  add(): void { if (this.artistsArray.length < this.maxArtists) this.artistsArray.push(this.artistControl()); }
  remove(i: number): void { if (this.artistsArray.length > 1) this.artistsArray.removeAt(i); }

  submit(): void {
    if (this.form.invalid) { this.form.markAllAsTouched(); return; }
    const artists = this.artistsArray.controls.map((c: FormControl<string | null>) => (c.value || '').trim()).filter(Boolean) as string[];
    if (!artists.length) { this.error.set('Enter at least one artist.'); return; }
    this.loading.set(true); this.error.set(null); this.matches.clear();
  const username = this.auth.user()?.username;
  const headers = username ? new HttpHeaders({ 'X-User-Username': username }) : undefined;
  this.http.post<MatchUser[]>(this.url('/findMusicMatches'), { artists }, { headers }).subscribe({
      next: (res: any) => { this.loading.set(false); this.matches.set(res); this.router.navigateByUrl('/matches'); },
      error: (err: any) => { this.loading.set(false); this.error.set(this.extractError(err) + ' Please try again later.'); }
    });
  this.auth.updateArtists(artists);
  }

  private url(path: string): string { return `${this.apiBase}${path}`; }

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
