import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormArray, FormBuilder, FormControl, Validators, FormGroup, AbstractControl, ValidationErrors, ValidatorFn } from '@angular/forms';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Router } from '@angular/router';
import { OnInit } from '@angular/core';
import { MatchService, MatchUser, Artist } from './match.service';
// import { environment } from '../environments/environment';
import { ApiBaseService } from './api-base.service';
import { AuthService } from './auth.service';
import { ArtistAutocompleteComponent } from './artist-autocomplete.component';
// Temporarily removed CDK drag-drop to fix build issues
// import { CdkDragDrop, CdkDrag, CdkDropList, moveItemInArray } from '@angular/cdk/drag-drop';

interface ArtistsFormShape { artists: FormArray<FormControl<string | null>>; }

@Component({
  selector: 'app-artists',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, ArtistAutocompleteComponent],
  template: `
  <section class="artists-section">
    <div class="page-header">
      <h2>Your Favorite Artists</h2>
      <p class="subtitle">List from most to least favorite. Add {{ minArtists }} to {{ maxArtists }} artists.</p>
      <p class="help-note">If you don't know what you listen to, you can use the <a href="https://www.statsforspotify.com/" target="_blank" rel="noopener noreferrer">Stats for Spotify</a> or <a href="https://statsmyapplemusic.com/" target="_blank" rel="noopener noreferrer">Stats for Apple Music</a> services.</p>
    </div>
    
    <form [formGroup]="form" (ngSubmit)="submit()" novalidate class="artists-form">
      <div class="form-card">
        <div class="form-header">
          <h3>Top {{ artistsArray.length }} Artist{{ artistsArray.length !== 1 ? 's' : '' }}</h3>
          <span class="count-badge">{{ artistsArray.length }}/{{ maxArtists }}</span>
        </div>
        
        <div formArrayName="artists" class="artists-list">
          <div class="artist-row" *ngFor="let ctrl of artistsArray.controls; let i = index">
            <!-- <div class="drag-handle" cdkDragHandle title="Drag to reorder">⋮⋮</div> -->
            <div class="input-wrapper">
              <span class="rank-number">{{ i + 1 }}</span>
              <app-artist-autocomplete
                [control]="ctrl"
                [placeholder]="'Enter artist #' + (i+1)"
                (artistSelected)="onArtistSelected($event, i)"
                class="artist-autocomplete">
              </app-artist-autocomplete>
              <button 
                type="button" 
                class="remove-btn" 
                (click)="remove(i)" 
                *ngIf="artistsArray.length > minArtists" 
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
        
        <!-- Duplicate artists error -->
        <div class="form-error" *ngIf="form.hasError('duplicateArtists')">
          <span class="error-icon">⚠️</span>
          <div>
            <strong>Duplicate artists detected!</strong><br>
            Please remove or change the duplicate entries. Each artist should only appear once in your list.
          </div>
        </div>
        
        <!-- Minimum artists error -->
        <div class="form-error" *ngIf="form.hasError('minArtists')">
          <span class="error-icon">⚠️</span>
          <div>
            <strong>Not enough artists!</strong><br>
            Please add at least {{ minArtists }} artists to continue.
          </div>
        </div>
      </div>
      
      <div class="submit-section">
        <button 
          type="submit" 
          class="submit-btn" 
          [disabled]="matches.loading() || form.invalid">
          <span *ngIf="!matches.loading()" class="btn-content">
            <span class="btn-icon">🎵</span>
            Find My Music Matches
          </span>
          <span *ngIf="matches.loading()" class="btn-content loading">
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

    .help-note {
      margin: 0.75rem 0 0 0;
      color: #6c757d;
      font-size: 0.9rem;
      font-style: italic;
    }

    .help-note a {
      color: #667eea;
      text-decoration: none;
      font-weight: 500;
      transition: color 0.2s ease;
    }

    .help-note a:hover {
      color: #764ba2;
      text-decoration: underline;
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

    .artist-row {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      transition: transform 0.2s ease;
    }

    .artist-row.cdk-drag-preview {
      box-shadow: 0 8px 25px rgba(0, 0, 0, 0.15);
      border-radius: 8px;
    }

    .artist-row.cdk-drag-animating {
      transition: transform 250ms cubic-bezier(0, 0, 0.2, 1);
    }

    .artists-list.cdk-drop-list-dragging .artist-row:not(.cdk-drag-placeholder) {
      transition: transform 250ms cubic-bezier(0, 0, 0.2, 1);
    }

    .drag-handle {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 2rem;
      height: 2rem;
      color: #6c757d;
      cursor: grab;
      font-size: 1rem;
      line-height: 1;
      user-select: none;
      transition: color 0.2s ease;
    }

    .drag-handle:hover {
      color: #495057;
    }

    .drag-handle:active {
      cursor: grabbing;
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

    .artist-autocomplete {
      flex: 1;
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

    .form-error {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      background: #f8d7da;
      color: #721c24;
      padding: 0.75rem;
      border-radius: 8px;
      margin-top: 1rem;
      border: 1px solid #f5c6cb;
    }

    .error-icon {
      font-size: 1rem;
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
export class ArtistsComponent implements OnInit {
  maxArtists = 20;
  minArtists = 5;
  error = signal<string | null>(null);
  form!: FormGroup<ArtistsFormShape>;
  constructor(private fb: FormBuilder, private http: HttpClient, public matches: MatchService, private router: Router, private auth: AuthService, private api: ApiBaseService) {
    // Initialize with 5 empty artist controls
    const initialControls = Array.from({ length: this.minArtists }, () => this.artistControl());
    this.form = this.fb.group<ArtistsFormShape>({ artists: this.fb.array(initialControls, { validators: [this.noDuplicatesValidator, this.minArtistsValidator] }) });
  }
  

  ngOnInit(): void {
    // Clear and reset form for new user session
    this.initializeForm();
  }

  private initializeForm(): void {
    // Clear existing form and error states
    const arr = this.artistsArray;
    while (arr.length) arr.removeAt(0);
    this.error.set(null);
    
    // Prefill for returning users
    const user = this.auth.user();
    if (user?.artists?.length) {
      // Add user's existing artists
      user.artists.slice(0, this.maxArtists).forEach((a: string) => 
        arr.push(this.fb.control<string | null>(a, [Validators.required, Validators.minLength(2), Validators.maxLength(240)]))
      );
      
      // Fill remaining slots up to minimum if needed
      while (arr.length < this.minArtists) {
        arr.push(this.artistControl());
      }
    } else {
      // Initialize with empty controls for new users
      for (let i = 0; i < this.minArtists; i++) {
        arr.push(this.artistControl());
      }
    }
  }

  // Custom validator to check for duplicate artists (case-insensitive)
  private noDuplicatesValidator = (control: AbstractControl): { [key: string]: any } | null => {
    const formArray = control as FormArray;
    const artists = formArray.controls
      .map((ctrl: AbstractControl) => (ctrl.value || '').toString().trim().toLowerCase())
      .filter((artist: string) => artist.length > 0);
    
    const duplicates = artists.filter((artist: string, index: number) => artists.indexOf(artist) !== index);
    return duplicates.length > 0 ? { duplicateArtists: { duplicates } } : null;
  };

  // Custom validator to check minimum number of artists
  private minArtistsValidator = (control: AbstractControl): { [key: string]: any } | null => {
    const formArray = control as FormArray;
    const validArtists = formArray.controls
      .map((ctrl: AbstractControl) => (ctrl.value || '').toString().trim())
      .filter((artist: string) => artist.length > 0);
    
    return validArtists.length < this.minArtists ? { minArtists: { required: this.minArtists, actual: validArtists.length } } : null;
  };

  get artistsArray(): FormArray<FormControl<string | null>> { return this.form.controls.artists; }
  private artistControl(): FormControl<string | null> { return this.fb.control<string | null>('', [Validators.required, Validators.minLength(2), Validators.maxLength(240)]); }

  onArtistSelected(artist: Artist, index: number): void {
    // Artist was selected from dropdown, the control value is already set
    console.log('Artist selected:', artist.name, 'at index:', index);
    // Trigger validation for duplicates immediately
    this.artistsArray.updateValueAndValidity();
  }

  // Helper method to get duplicate artist names for display
  getDuplicateArtists(): string[] {
    const duplicateErrors = this.form.get('artists')?.errors?.['duplicateArtists'];
    return duplicateErrors?.duplicates || [];
  }

  add(): void { 
    if (this.artistsArray.length < this.maxArtists) {
      this.artistsArray.push(this.artistControl());
      this.artistsArray.updateValueAndValidity();
    }
  }
  
  remove(i: number): void { 
    if (this.artistsArray.length > this.minArtists) {
      this.artistsArray.removeAt(i);
      this.artistsArray.updateValueAndValidity();
    }
  }

  // Temporarily commented out drag-drop functionality to fix build issues
  // drop(event: CdkDragDrop<FormControl<string | null>[]>): void {
  //   const artistsArray = this.artistsArray;
  //   
  //   // Get the current values
  //   const currentValues = artistsArray.controls.map((control: FormControl<string | null>) => control.value);
  //   
  //   // Move the values in the array
  //   moveItemInArray(currentValues, event.previousIndex, event.currentIndex);
  //   
  //   // Clear the form array
  //   while (artistsArray.length) {
  //     artistsArray.removeAt(0);
  //   }
  //   
  //   // Re-add the controls with the new order
  //   currentValues.forEach((value: string | null) => {
  //     const control = this.artistControl();
  //     control.setValue(value);
  //     artistsArray.push(control);
  //   });
  //   
  //   // Trigger validation
  //   artistsArray.updateValueAndValidity();
  // }

  submit(): void {
    if (this.form.invalid) { 
      this.form.markAllAsTouched();
      if (this.form.hasError('duplicateArtists')) {
        this.error.set('Please remove duplicate artists before submitting.');
        return;
      }
      if (this.form.hasError('minArtists')) {
        this.error.set(`Please add at least ${this.minArtists} artists before submitting.`);
        return;
      }
      return; 
    }
    const artists = this.artistsArray.controls.map((c: FormControl<string | null>) => (c.value || '').trim()).filter(Boolean) as string[];
    if (artists.length < this.minArtists) { 
      this.error.set(`Please add at least ${this.minArtists} artists.`); 
      return; 
    }
    
    // Additional duplicate check at submit time (case-insensitive)
    const lowerCaseArtists = artists.map(a => a.toLowerCase());
    const uniqueArtists = [...new Set(lowerCaseArtists)];
    if (lowerCaseArtists.length !== uniqueArtists.length) {
      this.error.set('Duplicate artists are not allowed. Please remove or change duplicate entries.');
      return;
    }
    
    this.matches.loading.set(true); this.error.set(null); this.matches.clear();
  const username = this.auth.user()?.username;
  const headers = username ? new HttpHeaders({ 'X-User-Username': username }) : undefined;
  const range = 'medium_term';
  const limit = 10;
  const qp = new URLSearchParams({ range, limit: limit.toString() }).toString();
  this.http.post<MatchUser[]>(this.api.url(`/findMusicMatches?${qp}`), { artists }, { headers }).subscribe({
      next: (res: any) => { this.matches.set(res); this.router.navigateByUrl('/matches'); },
      error: (err: any) => { this.matches.loading.set(false); this.error.set(this.extractError(err) + ' Please try again later.'); }
    });
  this.auth.updateArtists(artists);
  }

  // Deprecated helper retained for minimal changes in code paths above; prefer ApiBaseService
  private url(path: string): string { return this.api.url(path); }

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
