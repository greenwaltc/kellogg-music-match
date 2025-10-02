import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../environments/environment';

export type EventInterestType = 'INTERESTED' | 'GOING' | 'LOOKING_FOR_GROUP';

export interface SetEventInterestRequest {
  interestType: EventInterestType;
}

@Injectable({ providedIn: 'root' })
export class ConcertInterestService {
  private apiBase = window.__kmmConfig?.apiBaseUrl || environment.apiBaseUrl;

  constructor(private http: HttpClient) {}

  setInterest(eventId: string, interestType: EventInterestType) {
    const body: SetEventInterestRequest = { interestType };
    return this.http.post<void>(`${this.apiBase}/concerts/${encodeURIComponent(eventId)}/interest`, body, {
      headers: { 'X-User-Username': this.getUserIdentifier() }
    });
  }

  removeInterest(eventId: string) {
    return this.http.delete<void>(`${this.apiBase}/concerts/${encodeURIComponent(eventId)}/interest`, {
      headers: { 'X-User-Username': this.getUserIdentifier() }
    });
  }

  // Placeholder: actual user identity strategy can change (e.g., use auth token claims)
  private getUserIdentifier(): string {
    // Attempt to pull from local storage or a global user object later
    return localStorage.getItem('kmm_username') || 'anonymous-user';
  }
}
