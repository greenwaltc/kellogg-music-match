import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { MatchService } from './match.service';
import { SpotifyService } from './spotify.service';
import { NavStateService } from './nav-state.service';
import { PushService } from './services/push.service';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './home.component.html',
  styleUrls: ['./home.component.css']
})
export class HomeComponent {
  constructor(
    public matches: MatchService,
    private router: Router,
    private spotify: SpotifyService,
    private nav: NavStateService,
    public push: PushService,
  ) {}

  gotoMatches() { this.nav.markAllPrimaryVisited(); this.router.navigateByUrl('/matches'); }
  gotoEvents() { this.nav.markAllPrimaryVisited(); this.router.navigateByUrl('/events'); }
  async connectSpotify() { await this.spotify.beginAuth(); }

  async ngOnInit() {
    await this.push.refreshDeviceSubscriptionFlag();
  }

  notifPerm(): NotificationPermission {
    return (typeof Notification !== 'undefined') ? Notification.permission : 'default';
  }
}
