import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { ApiBaseService } from './api-base.service';

export interface EventsSearchFilters {
  keyword?: string;
  segmentName?: string;
  classificationName?: string;
  countryCode?: string;
  stateCode?: string;
  city?: string;
  latlong?: string; // "lat,long"
  radius?: number; // miles
  startDateTime?: string; // ISO 8601
  endDateTime?: string;   // ISO 8601
  sort?: string; // e.g. "date,asc"
  includeAssociated?: boolean;
}

export interface EventAssociationOverlay {
  myStatus?: 'INTERESTED' | 'GOING' | 'LFG' | 'NONE';
  interestedCount?: number;
  goingCount?: number;
  lfgCount?: number;
}

export interface EventDTO {
  id?: string; // local DB id if persisted
  externalId: string; // Ticketmaster id
  name: string;
  venueName?: string;
  city?: string;
  state?: string;
  country?: string;
  startUtc?: string;
  url?: string;
  association?: EventAssociationOverlay | null;
  // Allow unknown fields for forward compatibility
  [k: string]: any;
}

export interface EventsPage {
  page: number;
  size: number;
  total?: number; // total (TM only)
  items: EventDTO[];
}

@Injectable({ providedIn: 'root' })
export class EventsService {
  private http = inject(HttpClient);
  private api = inject(ApiBaseService);

  search(filters: EventsSearchFilters, page = 0, size = 20): Observable<EventsPage> {
    let params = new HttpParams()
      .set('page', String(page))
      .set('size', String(size));

    const add = (k: string, v: any) => {
      if (v === undefined || v === null || v === '') return;
      params = params.set(k, String(v));
    };

    add('keyword', filters.keyword);
    add('segmentName', filters.segmentName);
    add('classificationName', filters.classificationName);
    add('countryCode', filters.countryCode);
    add('stateCode', filters.stateCode);
    add('city', filters.city);
    add('latlong', filters.latlong);
    add('radius', filters.radius);
    add('startDateTime', filters.startDateTime);
    add('endDateTime', filters.endDateTime);
    add('sort', filters.sort);
    add('includeAssociated', filters.includeAssociated ?? true);

    return this.http.get<EventsPage>(this.api.url('/events/search'), { params });
  }

  setAssociation(eventId: string, status: 'INTERESTED'|'GOING'|'LFG'|'NONE') {
    const body = { status } as any;
    return this.http.post(this.api.url(`/events/${encodeURIComponent(eventId)}/association`), body);
  }

  clearAssociation(eventId: string) {
    return this.http.delete(this.api.url(`/events/${encodeURIComponent(eventId)}/association`));
  }
}
