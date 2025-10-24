import { Injectable } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class ApiBaseService {
  private readonly base = (typeof window !== 'undefined' && (window as any).__affyneConfig?.apiBaseUrl) || '/api';

  url(path: string): string {
    if (!path.startsWith('/')) return this.base + '/' + path;
    return this.base + path;
  }

  get baseUrl(): string { return this.base; }
}
