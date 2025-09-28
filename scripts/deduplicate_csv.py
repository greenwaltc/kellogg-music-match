#!/usr/bin/env python3
"""
Deduplicate MusicBrainz CSV file by removing exact duplicate rows.
Keeps the first occurrence of each unique row.
"""

import csv
import sys
from collections import OrderedDict

def deduplicate_csv(input_file, output_file):
    """Remove duplicate rows from CSV file."""
    
    seen_rows = OrderedDict()
    duplicate_count = 0
    total_rows = 0
    
    print(f"Reading from: {input_file}")
    print(f"Writing to: {output_file}")
    
    with open(input_file, 'r', encoding='utf-8') as infile:
        reader = csv.reader(infile)
        
        # Read header
        header = next(reader)
        
        # Process data rows
        for row in reader:
            total_rows += 1
            
            # Create a tuple from the row for hashing
            row_tuple = tuple(row)
            
            if row_tuple in seen_rows:
                duplicate_count += 1
                if duplicate_count <= 10:  # Show first 10 duplicates
                    print(f"Duplicate found: {row[1]} (ID: {row[0]})")
            else:
                seen_rows[row_tuple] = True
    
    print(f"\nProcessing complete:")
    print(f"  Total rows processed: {total_rows:,}")
    print(f"  Duplicates found: {duplicate_count:,}")
    print(f"  Unique rows: {len(seen_rows):,}")
    
    # Write deduplicated data
    with open(output_file, 'w', encoding='utf-8', newline='') as outfile:
        writer = csv.writer(outfile)
        
        # Write header
        writer.writerow(header)
        
        # Write unique rows
        for row_tuple in seen_rows.keys():
            writer.writerow(row_tuple)
    
    print(f"\nDeduplicated file saved to: {output_file}")
    return len(seen_rows), duplicate_count

def main():
    if len(sys.argv) != 3:
        print("Usage: python3 deduplicate_csv.py input.csv output.csv")
        sys.exit(1)
    
    input_file = sys.argv[1]
    output_file = sys.argv[2]
    
    try:
        unique_count, duplicate_count = deduplicate_csv(input_file, output_file)
        
        print(f"\nSummary:")
        print(f"  Original file: {input_file}")
        print(f"  Deduplicated file: {output_file}")
        print(f"  Removed {duplicate_count:,} duplicates")
        print(f"  Kept {unique_count:,} unique records")
        
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()