#!/usr/bin/env python3
import requests
import json
from datetime import datetime, timedelta
import sys

def debug_ticketmaster_api():
    # Use environment variables or defaults
    api_key = "test_key"  # This won't work, but we'll see the error structure
    
    # Calculate 6 months from now
    start_date = datetime.now()
    end_date = start_date + timedelta(days=6*30)  # Approximate 6 months
    
    # Build query parameters
    params = {
        'apikey': api_key,
        'classificationName': 'music',
        'city': 'Chicago',
        'stateCode': 'IL',
        'countryCode': 'US',
        'startDateTime': start_date.strftime('%Y-%m-%dT%H:%M:%SZ'),
        'endDateTime': end_date.strftime('%Y-%m-%dT%H:%M:%SZ'),
        'size': '200',
        'page': '0',
        'sort': 'date,asc',
        'includeSpellcheck': 'yes'
    }
    
    print(f"Querying Ticketmaster API for Chicago music events from {start_date.strftime('%Y-%m-%d')} to {end_date.strftime('%Y-%m-%d')}")
    print(f"API URL: https://app.ticketmaster.com/discovery/v2/events")
    print(f"Parameters: {json.dumps(params, indent=2)}")
    
    try:
        response = requests.get('https://app.ticketmaster.com/discovery/v2/events', params=params)
        print(f"\nResponse Status: {response.status_code}")
        
        if response.status_code == 401:
            print("Authentication failed - this is expected with test_key")
            print("Response:", response.text[:500])
        elif response.status_code == 200:
            data = response.json()
            print(f"Success! Found {data.get('page', {}).get('totalElements', 0)} total events")
            if '_embedded' in data and 'events' in data['_embedded']:
                events = data['_embedded']['events']
                print(f"Events in this page: {len(events)}")
                if events:
                    first_event = events[0]
                    last_event = events[-1]
                    print(f"First event date: {first_event.get('dates', {}).get('start', {}).get('dateTime', 'N/A')}")
                    print(f"Last event date: {last_event.get('dates', {}).get('start', {}).get('dateTime', 'N/A')}")
        else:
            print(f"Unexpected status: {response.status_code}")
            print("Response:", response.text[:500])
            
    except Exception as e:
        print(f"Error: {e}")

if __name__ == '__main__':
    debug_ticketmaster_api()