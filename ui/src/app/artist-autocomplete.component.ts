import { Component, Input, Output, EventEmitter, signal, inject, ElementRef, ViewChild, OnDestroy, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormControl, ReactiveFormsModule } from '@angular/forms';
import { MatchService, Artist } from './match.service';
import { debounceTime, distinctUntilChanged, switchMap, filter, catchError, takeUntil } from 'rxjs/operators';
import { of, Subject, from } from 'rxjs';

@Component({
  selector: 'app-artist-autocomplete',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  template: `
    <div class="autocomplete-container">
      <input 
        #searchInput
        [formControl]="control" 
        [placeholder]="placeholder"
        [maxlength]="240"
        class="autocomplete-input"
        (focus)="onFocus()"
        (blur)="onBlur()"
        (keydown)="onKeydown($event)"
      />
      <div class="char-counter" [class.warning]="control.value && control.value.length > 200">
        {{ control.value?.length || 0 }}/240
      </div>
      
      <div class="dropdown" *ngIf="showDropdown && (filteredArtists().length > 0 || isLoading() || error())" [class.show]="showDropdown">
        <div class="loading" *ngIf="isLoading()">
          <div class="spinner"></div>
          Searching artists...
        </div>
        
        <div class="error" *ngIf="error()">
          {{ error() }}
        </div>
        
        <div class="results" *ngIf="!isLoading() && !error()">
          <div 
            class="result-item" 
            *ngFor="let artist of filteredArtists(); let i = index"
            [class.highlighted]="i === highlightedIndex"
            (click)="selectArtist(artist)"
            (mouseenter)="highlightedIndex = i"
          >
            <span class="artist-name">{{ artist.name }}</span>
          </div>
          
          <div class="no-results" *ngIf="filteredArtists().length === 0 && searchTerm().length > 0">
            No artists found. You can still enter "{{ searchTerm() }}" as a new artist.
          </div>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .autocomplete-container {
      position: relative;
      width: 100%;
    }

    .autocomplete-input {
      width: 100%;
      padding: 0.75rem;
      border: 2px solid #e9ecef;
      border-radius: 8px;
      font-size: 1rem;
      background: #f8f9fa;
      transition: all 0.2s ease;
      outline: none;
    }

    .autocomplete-input:focus {
      background: #fff;
      border-color: #667eea;
      box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
    }

    .char-counter {
      position: absolute;
      right: 0.75rem;
      top: 50%;
      transform: translateY(-50%);
      font-size: 0.75rem;
      color: #6c757d;
      pointer-events: none;
    }

    .char-counter.warning {
      color: #dc3545;
      font-weight: 600;
    }

    .dropdown {
      position: absolute;
      top: 100%;
      left: 0;
      right: 0;
      background: white;
      border: 1px solid #e0e0e0;
      border-radius: 8px;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
      z-index: 1000;
      max-height: 300px;
      overflow-y: auto;
      opacity: 0;
      transform: translateY(-10px);
      transition: all 0.2s ease;
    }

    .dropdown.show {
      opacity: 1;
      transform: translateY(0);
    }

    .loading, .error {
      padding: 1rem;
      text-align: center;
      color: #6c757d;
    }

    .loading {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 0.5rem;
    }

    .spinner {
      width: 1rem;
      height: 1rem;
      border: 2px solid #e9ecef;
      border-top: 2px solid #667eea;
      border-radius: 50%;
      animation: spin 1s linear infinite;
    }

    @keyframes spin {
      to { transform: rotate(360deg); }
    }

    .error {
      color: #dc3545;
    }

    .result-item {
      padding: 0.75rem 1rem;
      cursor: pointer;
      border-bottom: 1px solid #f0f0f0;
      transition: background-color 0.15s ease;
    }

    .result-item:last-child {
      border-bottom: none;
    }

    .result-item:hover,
    .result-item.highlighted {
      background-color: #f8f9fa;
    }

    .result-item.highlighted {
      background-color: #e7f1ff;
    }

    .artist-name {
      font-weight: 500;
      color: #2c3e50;
    }

    .no-results {
      padding: 1rem;
      text-align: center;
      color: #6c757d;
      font-style: italic;
    }
  `]
})
export class ArtistAutocompleteComponent implements OnInit, OnDestroy {
  @Input() control!: FormControl<string | null>;
  @Input() placeholder: string = 'Search for an artist...';
  @Output() artistSelected = new EventEmitter<Artist>();
  
  @ViewChild('searchInput') searchInput!: ElementRef;
  
  private matchService = inject(MatchService);
  private destroy$ = new Subject<void>();
  
  filteredArtists = signal<Artist[]>([]);
  isLoading = signal(false);
  error = signal<string>('');
  searchTerm = signal<string>('');
  showDropdown = false;
  highlightedIndex = -1;

  ngOnInit() {
    // Set up search functionality with debouncing
    this.control.valueChanges.pipe(
      debounceTime(300),
      distinctUntilChanged(),
      filter((value: string | null) => (value?.trim()?.length || 0) >= 2),
      switchMap((query: string | null) => {
        if (!query || query.trim().length < 2) {
          return of([]);
        }
        
        this.isLoading.set(true);
        this.error.set('');
        this.searchTerm.set(query.trim());
        
        return from(this.matchService.searchArtists(query.trim(), 10)).pipe(
          catchError((err: any) => {
            this.isLoading.set(false);
            this.error.set('Failed to search artists');
            return of([]);
          })
        );
      }),
      takeUntil(this.destroy$),
      catchError(() => {
        this.isLoading.set(false);
        this.error.set('Failed to search artists');
        return of([]);
      })
    ).subscribe((artists: Artist[]) => {
      this.isLoading.set(false);
      this.filteredArtists.set(artists);
      this.highlightedIndex = -1;
    });
  }

  ngOnDestroy() {
    this.destroy$.next();
    this.destroy$.complete();
  }

  onFocus() {
    this.showDropdown = true;
  }

  onBlur() {
    // Delay hiding to allow click events to fire
    setTimeout(() => {
      this.showDropdown = false;
    }, 200);
  }

  onKeydown(event: KeyboardEvent) {
    const artists = this.filteredArtists();
    
    switch (event.key) {
      case 'ArrowDown':
        event.preventDefault();
        this.highlightedIndex = Math.min(this.highlightedIndex + 1, artists.length - 1);
        break;
      case 'ArrowUp':
        event.preventDefault();
        this.highlightedIndex = Math.max(this.highlightedIndex - 1, -1);
        break;
      case 'Enter':
        event.preventDefault();
        if (this.highlightedIndex >= 0 && this.highlightedIndex < artists.length) {
          this.selectArtist(artists[this.highlightedIndex]);
        }
        break;
      case 'Escape':
        this.showDropdown = false;
        this.searchInput.nativeElement.blur();
        break;
    }
  }

  selectArtist(artist: Artist) {
    this.control.setValue(artist.name);
    this.artistSelected.emit(artist);
    this.showDropdown = false;
    this.highlightedIndex = -1;
  }
}