import { Injectable } from '@angular/core';
import { environment } from '../environments/environment';

@Injectable({ providedIn: 'root' })
export class ApiBaseService {
  private readonly base = window.__kmmConfig?.apiBaseUrl || environment.apiBaseUrl;

  url(path: string): string {
    if (!path.startsWith('/')) return this.base + '/' + path;
    return this.base + path;
  }

  get baseUrl(): string { return this.base; }
}
