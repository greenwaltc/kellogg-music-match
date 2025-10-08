import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SpotifyConnectComponent } from './spotify-connect.component';

// Provide a mock SpotifyService with beginAuth spy
class MockSpotifyService {
  beginAuth = jasmine.createSpy('beginAuth').and.returnValue(Promise.resolve());
}

describe('SpotifyConnectComponent', () => {
  let component: SpotifyConnectComponent;
  let fixture: ComponentFixture<SpotifyConnectComponent>;
  let mockService: MockSpotifyService;

  beforeEach(async () => {
    mockService = new MockSpotifyService();
    await TestBed.configureTestingModule({
      imports: [SpotifyConnectComponent],
      providers: [{ provide: (await import('./spotify.service')).SpotifyService, useValue: mockService }]
    }).compileComponents();

    fixture = TestBed.createComponent(SpotifyConnectComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('emits connected after successful connect()', async () => {
    const spy = jasmine.createSpy('connected');
    component.connected.subscribe(spy);
    await component.connect();
    expect(mockService.beginAuth).toHaveBeenCalled();
    expect(spy).toHaveBeenCalled();
    expect(component.loading).toBeFalse();
  });

  it('prevents double invocation while loading', async () => {
    // make beginAuth hang until we resolve manually
    let resolveFn: () => void; let promise = new Promise<void>(res => resolveFn = res);
    mockService.beginAuth.and.returnValue(promise);
    component.connect();
    component.connect(); // second call should be ignored
    expect(mockService.beginAuth).toHaveBeenCalledTimes(1);
    // finish
    resolveFn!();
    await promise;
    expect(component.loading).toBeFalse();
  });
});
