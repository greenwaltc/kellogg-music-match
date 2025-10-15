import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { MatchesComponent } from './matches.component';
import { MatchService, MatchUser } from './match.service';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { AuthService } from './auth.service';
import { ApiBaseService } from './api-base.service';
import { SpotifyService } from './spotify.service';

class MockAuthService { user = () => ({ username: 'alice', artists: [] }); }
class MockApiBase { baseUrl = 'http://test'; url(p:string){return this.baseUrl+p;} }
class MockSpotifyService { beginAuth = jasmine.createSpy('beginAuth').and.returnValue(Promise.resolve()); }

// Extend MatchService to expose forcing readiness in tests without going through callback component
class TestMatchService extends MatchService {}

describe('MatchesComponent integration-ish flow', () => {
  let fixture: ComponentFixture<MatchesComponent>;
  let component: MatchesComponent;
  let matchService: MatchService;
  let http: HttpTestingController;

  beforeEach(async () => {
  localStorage.clear();
    await TestBed.configureTestingModule({
      imports:[HttpClientTestingModule, MatchesComponent],
      providers:[
        { provide: AuthService, useClass: MockAuthService },
        { provide: ApiBaseService, useClass: MockApiBase },
        { provide: SpotifyService, useClass: MockSpotifyService },
        MatchService
      ]
    }).compileComponents();
    fixture = TestBed.createComponent(MatchesComponent);
    component = fixture.componentInstance;
    matchService = TestBed.inject(MatchService);
    http = TestBed.inject(HttpTestingController);
    fixture.detectChanges();
  });

  it('shows pre-connect placeholder before readiness', () => {
    const el: HTMLElement = fixture.nativeElement;
    expect(el.querySelector('.pre-connect-placeholder')).toBeTruthy();
  expect(el.textContent || '').toContain('Connect Spotify');
  });

  it('fetches matches after markSpotifyReadyAndRefetch and hides placeholder', () => {
    matchService.markSpotifyReadyAndRefetch('medium_term');
    fixture.detectChanges();
    const req = http.expectOne(r => r.url.startsWith('http://test/findMusicMatches?range=medium_term&limit=50'));
    expect(req.request.method).toBe('POST');
    req.flush(<MatchUser[]>[]);
    fixture.detectChanges();
    const el: HTMLElement = fixture.nativeElement;
    expect(el.querySelector('.pre-connect-placeholder')).toBeFalsy();
  });

  it('forces fresh fetch on recent range change (cache-bust param present)', fakeAsync(() => {
    // mark ready and flush first fetch
    matchService.markSpotifyReadyAndRefetch('medium_term');
    let req = http.expectOne(r => r.url.startsWith('http://test/findMusicMatches?range=medium_term&limit=50'));
    req.flush([]);
    fixture.detectChanges();
    // Change range quickly; expect fresh param appended
    component.selectRange('short_term');
    tick(200); // allow debounce
    req = http.expectOne(r => r.url.includes('range=short_term') && r.url.includes('fresh='));
    expect(req).toBeTruthy();
    req.flush([]);
  }));

  it('wires name filter input to userName query param (debounced)', fakeAsync(() => {
    // Flush any initial status probe
    const pending = http.match(() => true);
    pending.forEach(r => r.flush({}));
    // Mark ready; we don't assert the initial POST here
    matchService.markSpotifyReadyAndRefetch('medium_term');
    http.match(r => r.url.startsWith('http://test/findMusicMatches')).forEach(r => r.flush([]));
    fixture.detectChanges();

    // Type into name filter
    component.onNameFilterChange('Jane');
    tick(450); // allow 400ms debounce + buffer
  let req = http.expectOne(r => r.url.includes('userName=Jane'));
    req.flush([]);

    // Update to full name
    component.onNameFilterChange('Jane Doe');
    tick(450); // allow 400ms debounce + buffer
  req = http.expectOne(r => r.url.includes('userName=Jane+Doe'));
    req.flush([]);

    // Clear filter
    component.onNameFilterChange('   ');
    tick(450); // allow 400ms debounce + buffer
    req = http.expectOne(r => r.url.startsWith('http://test/findMusicMatches?') && !r.url.includes('userName='));
    req.flush([]);
    expect(true).toBeTrue();
  }));
});
