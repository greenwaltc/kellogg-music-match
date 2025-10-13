import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { MatchService } from './match.service';
import { SpotifyService } from './spotify.service';
import { NavStateService } from './nav-state.service';

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
  ) {}

  gotoMatches() { this.nav.markAllPrimaryVisited(); this.router.navigateByUrl('/matches'); }
  gotoEvents() { this.nav.markAllPrimaryVisited(); this.router.navigateByUrl('/chicago-events'); }
  async connectSpotify() { await this.spotify.beginAuth(); }
}
