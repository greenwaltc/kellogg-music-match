#!/usr/bin/env python3
"""
Fetch all artists from MusicBrainz API with pagination and save to file.
MusicBrainz API documentation: https://musicbrainz.org/doc/MusicBrainz_API
"""

import requests
import json
import time
import csv
import argparse
from datetime import datetime
from typing import List, Dict, Any

class MusicBrainzArtistFetcher:
    def __init__(self, delay: float = 1.0):
        self.base_url = "https://musicbrainz.org/ws/2"
        self.delay = delay  # Delay between requests to be respectful
        self.headers = {
            'User-Agent': 'KelloggMusicMatch/1.0 (cameron@greenwalt.app)',
            'Accept': 'application/json'
        }
        
    def fetch_artists_page(self, offset: int = 0, limit: int = 100) -> Dict[str, Any]:
        """Fetch a single page of artists from MusicBrainz API."""
        url = f"{self.base_url}/artist"
        params = {
            'query': '*',
            'fmt': 'json',
            'limit': limit,
            'offset': offset
        }
        
        try:
            response = requests.get(url, params=params, headers=self.headers)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"Error fetching page at offset {offset}: {e}")
            return {}
    
    def extract_artist_data(self, artist: Dict[str, Any]) -> Dict[str, Any]:
        """Extract relevant artist data from MusicBrainz response."""
        return {
            'id': artist.get('id', ''),
            'name': artist.get('name', ''),
            'sort_name': artist.get('sort-name', ''),
            'type': artist.get('type', ''),
            'gender': artist.get('gender', ''),
            'country': artist.get('country', ''),
            'life_span_begin': artist.get('life-span', {}).get('begin', ''),
            'life_span_end': artist.get('life-span', {}).get('end', ''),
            'disambiguation': artist.get('disambiguation', ''),
            'score': artist.get('score', 0)
        }
    
    def fetch_all_artists(self, max_artists: int = None, output_format: str = 'json') -> List[Dict[str, Any]]:
        """Fetch all artists with pagination."""
        all_artists = []
        offset = 0
        limit = 100
        total_fetched = 0
        
        print(f"Starting to fetch artists from MusicBrainz API...")
        print(f"Using delay of {self.delay} seconds between requests")
        
        while True:
            print(f"Fetching page: offset={offset}, limit={limit}")
            
            data = self.fetch_artists_page(offset, limit)
            if not data or 'artists' not in data:
                print("No more data available or error occurred")
                break
            
            artists = data['artists']
            if not artists:
                print("No artists in response")
                break
            
            # Process artists from this page
            for artist in artists:
                artist_data = self.extract_artist_data(artist)
                all_artists.append(artist_data)
                total_fetched += 1
                
                if max_artists and total_fetched >= max_artists:
                    print(f"Reached maximum limit of {max_artists} artists")
                    return all_artists
            
            print(f"Fetched {len(artists)} artists (total: {total_fetched})")
            
            # Check if we've reached the end
            if len(artists) < limit:
                print("Reached end of results")
                break
            
            offset += limit
            
            # Be respectful to the API
            time.sleep(self.delay)
        
        print(f"Finished fetching {total_fetched} artists")
        return all_artists
    
    def save_to_json(self, artists: List[Dict[str, Any]], filename: str):
        """Save artists to JSON file."""
        with open(filename, 'w', encoding='utf-8') as f:
            json.dump({
                'metadata': {
                    'total_artists': len(artists),
                    'fetched_at': datetime.now().isoformat(),
                    'source': 'MusicBrainz API'
                },
                'artists': artists
            }, f, indent=2, ensure_ascii=False)
        print(f"Saved {len(artists)} artists to {filename}")
    
    def save_to_csv(self, artists: List[Dict[str, Any]], filename: str):
        """Save artists to CSV file."""
        if not artists:
            print("No artists to save")
            return
        
        fieldnames = artists[0].keys()
        with open(filename, 'w', newline='', encoding='utf-8') as f:
            writer = csv.DictWriter(f, fieldnames=fieldnames)
            writer.writeheader()
            writer.writerows(artists)
        print(f"Saved {len(artists)} artists to {filename}")

def main():
    parser = argparse.ArgumentParser(description='Fetch artists from MusicBrainz API')
    parser.add_argument('--max-artists', type=int, help='Maximum number of artists to fetch')
    parser.add_argument('--delay', type=float, default=1.0, help='Delay between requests (seconds)')
    parser.add_argument('--format', choices=['json', 'csv'], default='json', help='Output format')
    parser.add_argument('--output', type=str, help='Output filename (auto-generated if not provided)')
    
    args = parser.parse_args()
    
    # Create fetcher
    fetcher = MusicBrainzArtistFetcher(delay=args.delay)
    
    # Fetch artists
    artists = fetcher.fetch_all_artists(max_artists=args.max_artists)
    
    if not artists:
        print("No artists fetched")
        return
    
    # Generate filename if not provided
    if args.output:
        filename = args.output
    else:
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        filename = f"musicbrainz_artists_{timestamp}.{args.format}"
    
    # Save to file
    if args.format == 'json':
        fetcher.save_to_json(artists, filename)
    else:
        fetcher.save_to_csv(artists, filename)

if __name__ == "__main__":
    main()