import { TestBed } from '@angular/core/testing';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { ConcertInterestService } from './concert-interest.service';
import { ApiBaseService } from './api-base.service';

class ApiBaseServiceStub {
  private readonly apiBaseUrl = 'http://test-api';
  url(path: string) { return path.startsWith('/') ? this.apiBaseUrl + path : this.apiBaseUrl + '/' + path; }
  get baseUrl() { return this.apiBaseUrl; }
}

describe('ConcertInterestService', () => {
  let service: ConcertInterestService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    localStorage.setItem('kmm_token', 'header.' + btoa(JSON.stringify({ username: 'alice' })) + '.sig');

    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [
        ConcertInterestService,
        { provide: ApiBaseService, useClass: ApiBaseServiceStub }
      ]
    });

    service = TestBed.inject(ConcertInterestService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    localStorage.removeItem('kmm_token');
  });

  it('should POST set interest with body only (auth handled by interceptor)', () => {
    service.setInterest('evt1', 'GOING').subscribe();

    const req = httpMock.expectOne('http://test-api/concerts/evt1/interest');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ interestType: 'GOING' });
    req.flush({});
  });

  it('should DELETE interest without legacy header', () => {
    service.removeInterest('evt2').subscribe();

    const req = httpMock.expectOne('http://test-api/concerts/evt2/interest');
    expect(req.request.method).toBe('DELETE');
    req.flush({});
  });

  it('should still send request even if token missing (API will 401)', () => {
    localStorage.removeItem('kmm_token');
    service.setInterest('evt3', 'INTERESTED').subscribe({
      error: () => {}
    });
    const req = httpMock.expectOne('http://test-api/concerts/evt3/interest');
    expect(req.request.method).toBe('POST');
    req.flush({}, { status: 401, statusText: 'Unauthorized' });
  });
});
