import { Injectable } from '@angular/core';
import { ApiBaseService } from './api-base.service';
import { HttpClient } from '@angular/common/http';
import { Observable, from } from 'rxjs';
import { switchMap } from 'rxjs/operators';
import { ConfigService } from './config.service';

// Lightweight PKCE + state handling for Spotify Authorization Code with PKCE.
// For now we just build the URL and redirect the browser.

@Injectable({ providedIn: 'root' })
export class SpotifyService {
  // Fallbacks if runtime config omits them.
  private fallbackRedirectUri = window.location.origin + '/spotify/callback';
  private fallbackClientId = '';
  private scope = [
    'user-read-email',
    'user-read-private',
    'user-top-read',
    'playlist-read-private'
  ].join(' ');

  constructor(private api: ApiBaseService, private http: HttpClient, private cfg: ConfigService) {}

  // Generate a random string (state & code verifier)
  private randomString(length: number): string {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let result = '';
    const array = new Uint32Array(length);
    crypto.getRandomValues(array);
    for (let i = 0; i < length; i++) {
      result += chars[array[i] % chars.length];
    }
    return result;
  }

  private async sha256(plain: string): Promise<ArrayBuffer> {
    const encoder = new TextEncoder();
    const data = encoder.encode(plain);
    return await crypto.subtle.digest('SHA-256', data);
  }

  private base64UrlEncode(arrayBuffer: ArrayBuffer): string {
    const bytes = new Uint8Array(arrayBuffer);
    let str = '';
    bytes.forEach(b => str += String.fromCharCode(b));
    return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  }

  async beginAuth(): Promise<void> {
  const state = this.randomString(16);
    const codeVerifier = this.randomString(64);
    localStorage.setItem('spotify_auth_state', state);
    localStorage.setItem('spotify_code_verifier', codeVerifier);
    const codeChallenge = this.base64UrlEncode(await this.sha256(codeVerifier));

    // Explicit, single encoding to avoid both no-encoding and double-encoding scenarios.
    const cfg = await this.cfg.getConfig();
    const redirectUri = cfg.spotifyRedirectUri && cfg.spotifyRedirectUri.trim().length > 0 ? cfg.spotifyRedirectUri : this.fallbackRedirectUri;
    const clientId = cfg.spotifyClientId && cfg.spotifyClientId.trim().length > 0 ? cfg.spotifyClientId : this.fallbackClientId;

    const params = new URLSearchParams({
      response_type: 'code',
      client_id: clientId,
      scope: this.scope,
      // Provide raw redirect URI; URLSearchParams will encode it exactly once.
      redirect_uri: redirectUri,
      state,
      code_challenge_method: 'S256',
      code_challenge: codeChallenge
    });
    // IMPORTANT: Do NOT manually encode again; previous version caused double encoding (%253A pattern)
    const authUrl = 'https://accounts.spotify.com/authorize?' + params.toString();
    console.debug('[SpotifyAuth] Redirecting to', authUrl, 'using redirectUri=', redirectUri);
    window.location.href = authUrl;
  }

  // Exchange code for token via backend (backend will hold client secret)
  exchangeCode(code: string, state: string): Observable<any> {
    const storedState = localStorage.getItem('spotify_auth_state');
    if (!storedState || storedState !== state) {
      return from(Promise.reject(new Error('State mismatch')));
    }
    const codeVerifier = localStorage.getItem('spotify_code_verifier') || '';
    return from(this.cfg.getConfig()).pipe(
      switchMap(cfg => {
        const redirectUri = cfg.spotifyRedirectUri && cfg.spotifyRedirectUri.trim().length > 0 ? cfg.spotifyRedirectUri : this.fallbackRedirectUri;
        return this.http.post(this.api.url('/spotify/oauth/callback'), { code, codeVerifier, redirectUri });
      })
    );
  }
}