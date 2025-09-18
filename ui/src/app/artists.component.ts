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
  <form [formGroup]="form" (ngSubmit)="submit()" novalidate>
    <h2>Your Favorite Artists (Top {{ artistsArray.length }})</h2>
    <p>List from most to least favorite. Add up to {{ maxArtists }}.</p>
    <div formArrayName="artists">
      <div class="artist-row" *ngFor="let ctrl of artistsArray.controls; let i = index">
        <input [formControlName]="i" type="text" [attr.placeholder]="'Artist #' + (i+1)" />
        <button type="button" class="remove" (click)="remove(i)" *ngIf="artistsArray.length > 1" aria-label="Remove">×</button>
      </div>
    </div>
    <div class="add-row">
      <button type="button" (click)="add()" [disabled]="artistsArray.length >= maxArtists">＋</button>
      <span *ngIf="artistsArray.length >= maxArtists" class="limit">Max reached</span>
    </div>
    <button type="submit" [disabled]="loading()">{{ loading() ? 'Finding Matches...' : 'Find Matches' }}</button>
    <div class="error" *ngIf="error()">{{ error() }}</div>
  </form>
  `
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
      user.artists.slice(0, this.maxArtists).forEach(a => arr.push(this.fb.control<string | null>(a, [Validators.required, Validators.minLength(2)])));
    }
  }

  get artistsArray(): FormArray<FormControl<string | null>> { return this.form.controls.artists; }
  private artistControl(): FormControl<string | null> { return this.fb.control<string | null>('', [Validators.required, Validators.minLength(2)]); }

  add(): void { if (this.artistsArray.length < this.maxArtists) this.artistsArray.push(this.artistControl()); }
  remove(i: number): void { if (this.artistsArray.length > 1) this.artistsArray.removeAt(i); }

  submit(): void {
    if (this.form.invalid) { this.form.markAllAsTouched(); return; }
    const artists = this.artistsArray.controls.map(c => (c.value || '').trim()).filter(Boolean) as string[];
    if (!artists.length) { this.error.set('Enter at least one artist.'); return; }
    this.loading.set(true); this.error.set(null); this.matches.clear();
  const username = this.auth.user()?.username;
  const headers = username ? new HttpHeaders({ 'X-User-Username': username }) : undefined;
  this.http.post<MatchUser[]>(this.url('/findMusicMatches'), { artists }, { headers }).subscribe({
      next: (res) => { this.loading.set(false); this.matches.set(res); this.router.navigateByUrl('/matches'); },
      error: (err) => { this.loading.set(false); this.error.set(this.extractError(err) + ' Please try again later.'); }
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
