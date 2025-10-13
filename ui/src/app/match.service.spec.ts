import { TestBed } from '@angular/core/testing';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { MatchService } from './match.service';
import { AuthService } from './auth.service';
import { ApiBaseService } from './api-base.service';

class MockAuthService { user = () => ({ username: 'alice', artists: [] }); }
class MockApiBase { baseUrl = 'http://test'; url(p:string){return this.baseUrl+p;} }

describe('MatchService spotify gating', () => {
  let svc: MatchService;
  let http: HttpTestingController;

  beforeEach(() => {
    localStorage.removeItem('kmmSpotifyReady');
    localStorage.removeItem('kmmSpotifyReadyTs');
    TestBed.configureTestingModule({
      imports:[HttpClientTestingModule],
      providers:[
        MatchService,
        { provide: AuthService, useClass: MockAuthService},
        { provide: ApiBaseService, useClass: MockApiBase }
      ]
    });
    svc = TestBed.inject(MatchService);
    http = TestBed.inject(HttpTestingController);
  });

  it('does not fetch before spotifyReady', () => {
    svc.fetchIfReady();
    // Expect no request matching base pattern (fresh param absent because not called)
    http.expectNone(req => req.url.startsWith('http://test/findMusicMatches?range=medium_term&limit=50'));
    expect(svc.matches()).toBeNull();
  });

  it('fetches after markSpotifyReadyAndRefetch (with fresh param)', () => {
    svc.markSpotifyReadyAndRefetch('medium_term');
    const req = http.expectOne(r => r.url.startsWith('http://test/findMusicMatches?')
      && r.url.includes('range=medium_term')
      && r.url.includes('limit=50')
      && r.url.includes('fresh='));
    expect(req.request.method).toBe('POST');
    req.flush([]);
  });
});
