import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { ApiBaseService } from './api-base.service';

export type EventInterestType = 'INTERESTED' | 'GOING' | 'LOOKING_FOR_GROUP';

export interface SetEventInterestRequest {
  interestType: EventInterestType;
}

@Injectable({ providedIn: 'root' })
export class ConcertInterestService {
  constructor(private http: HttpClient, private api: ApiBaseService) {}

  setInterest(eventId: string, interestType: EventInterestType) {
    const body: SetEventInterestRequest = { interestType };
    return this.http.post<void>(`${this.api.url('/concerts/')}${encodeURIComponent(eventId)}/interest`, body);
  }

  removeInterest(eventId: string) {
    return this.http.delete<void>(`${this.api.url('/concerts/')}${encodeURIComponent(eventId)}/interest`);
  }
}
