import { TestBed } from '@angular/core/testing';
import { PushService } from './push.service';
import { SwPush } from '@angular/service-worker';
import { Subject } from 'rxjs';
import { HttpClientTestingModule } from '@angular/common/http/testing';
import { AppConfigService } from './app-config.service';
import { ApiBaseService } from '../api-base.service';
import { ToastService } from './toast.service';

// Minimal mocks for Notification and ServiceWorker contexts
function mockNotification(permission: NotificationPermission) {
  (globalThis as any).Notification = ({ permission } as unknown) as Notification;
}

function mockSW(hasSub: boolean) {
  const value = {
    ready: Promise.resolve({
      pushManager: {
        getSubscription: () => Promise.resolve(hasSub ? ({ endpoint: 'x' } as any) : null)
      }
    })
  } as any;
  Object.defineProperty(navigator, 'serviceWorker', { value, configurable: true });
}

describe('PushService.refreshDeviceSubscriptionFlag', () => {
  beforeEach(() => {
    const messages$ = new Subject<any>();
    const clicks$ = new Subject<any>();
    TestBed.configureTestingModule({ 
      imports: [HttpClientTestingModule],
      providers: [
        PushService,
        { provide: SwPush, useValue: { isEnabled: true, requestSubscription: () => Promise.resolve({} as any), messages: messages$.asObservable(), notificationClicks: clicks$.asObservable() } },
        { provide: AppConfigService, useValue: { get: () => ({ vapidPublicKey: 'B'.repeat(65) }) } },
        { provide: ApiBaseService, useValue: { url: (p: string) => p } },
        { provide: ToastService, useValue: { show: () => {} } },
      ]
    });
  });

  it('sets false when Notification is not supported', async () => {
    (globalThis as any).Notification = undefined;
    const svc = TestBed.inject(PushService);
    await svc.refreshDeviceSubscriptionFlag();
    expect(svc.hasDeviceSubscription()).toBeFalse();
  });

  it('sets false when permission is not granted', async () => {
    mockNotification('denied');
    mockSW(false);
    const svc = TestBed.inject(PushService);
    await svc.refreshDeviceSubscriptionFlag();
    expect(svc.hasDeviceSubscription()).toBeFalse();
  });

  it('sets true when permission granted and subscription exists', async () => {
    mockNotification('granted');
    mockSW(true);
    const svc = TestBed.inject(PushService);
    await svc.refreshDeviceSubscriptionFlag();
    expect(svc.hasDeviceSubscription()).toBeTrue();
  });

  it('sets false when permission granted but no subscription', async () => {
    mockNotification('granted');
    mockSW(false);
    const svc = TestBed.inject(PushService);
    await svc.refreshDeviceSubscriptionFlag();
    expect(svc.hasDeviceSubscription()).toBeFalse();
  });
});
