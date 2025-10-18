import { ComponentFixture, TestBed } from '@angular/core/testing';
import { HomeComponent } from './home.component';
import { MatchService } from './match.service';
import { SpotifyService } from './spotify.service';
import { Router } from '@angular/router';
import { PushService } from './services/push.service';
import { SwPush } from '@angular/service-worker';
import { Subject } from 'rxjs';
import { NavStateService } from './nav-state.service';

class MockMatchService { spotifyReady = () => false as any; }
class MockSpotifyService { beginAuth = jasmine.createSpy('beginAuth').and.returnValue(Promise.resolve()); }
class MockRouter { navigateByUrl = jasmine.createSpy('navigateByUrl'); }

describe('HomeComponent', () => {
  let fixture: ComponentFixture<HomeComponent>;
  let component: HomeComponent;
  let nav: NavStateService;
  let router: MockRouter;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [HomeComponent],
      providers: [
        { provide: MatchService, useClass: MockMatchService },
        { provide: SpotifyService, useClass: MockSpotifyService },
        { provide: Router, useClass: MockRouter },
        { provide: PushService, useValue: { hasDeviceSubscription: () => false, refreshDeviceSubscriptionFlag: () => Promise.resolve(), ensureSubscribed: () => Promise.resolve(null) } },
        { provide: SwPush, useValue: { isEnabled: true, messages: new Subject().asObservable(), notificationClicks: new Subject().asObservable() } },
        NavStateService,
      ]
    }).compileComponents();
    fixture = TestBed.createComponent(HomeComponent);
    component = fixture.componentInstance;
    nav = TestBed.inject(NavStateService);
    router = TestBed.inject(Router) as unknown as MockRouter;
    fixture.detectChanges();
  });

  it('marks both menus visited when clicking Matches', () => {
    component.gotoMatches();
    expect(nav.visitedMatches()).toBeTrue();
    expect(nav.visitedEvents()).toBeTrue();
    expect(router.navigateByUrl).toHaveBeenCalledWith('/matches');
  });

  it('marks both menus visited when clicking Events', () => {
    component.gotoEvents();
    expect(nav.visitedMatches()).toBeTrue();
    expect(nav.visitedEvents()).toBeTrue();
    expect(router.navigateByUrl).toHaveBeenCalledWith('/events');
  });
});
