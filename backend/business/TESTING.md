# Music Matching Behavioral Tests

This directory contains comprehensive behavioral tests for the music matching system using the Ginkgo testing framework.

## Test Coverage

The behavioral tests verify the following aspects of the music matching algorithm:

### Core Matching Scenarios
- **Identical Preferences**: Users with exactly the same artist preferences should get maximum similarity scores
- **Different Preferences**: Users with different artist preferences should get lower similarity scores
- **No Overlap**: Users with completely different preferences should get no matches
- **Position Sensitivity**: Users with similar artists in different orders should get position-adjusted similarity scores

### Edge Cases
- Empty artist lists
- Single artist preferences
- Caller exclusion from results
- Whitespace and case normalization

### Algorithm Validation
- **PWO Distance Calculation**: Validates that the engine correctly calculates Position-Weighted Overlap distances
- **Database Function Alignment**: Tests scenarios that correspond to the PostgreSQL `pwo_distance` function outcomes:
  - Distance = 0.0 (identical arrays) → Maximum similarity scores
  - Distance = 1.0 (no overlap) → No matches returned
  - Position-sensitive scoring → Adjusted similarity based on artist order

## Running the Tests

### Prerequisites
Make sure you have Ginkgo installed:
```bash
go install github.com/onsi/ginkgo/v2/ginkgo
```

### Run All Tests
```bash
cd backend/business
~/go/bin/ginkgo run .
```

### Run Tests with Verbose Output
```bash
cd backend/business  
~/go/bin/ginkgo run -v .
```

### Run Specific Test Context
```bash
cd backend/business
~/go/bin/ginkgo run --focus "when target user has identical preferences" .
```

## Test Structure

The tests are organized into two main describe blocks:

1. **Music Matching Engine Behavior**: Tests the actual `ComputeMatches` method behavior
2. **Algorithm Validation Against PWO Distance Function**: Validates alignment with PostgreSQL PWO similarity calculations

Each test creates specific user scenarios and validates that the matching engine produces expected results for similarity scores, overlap counts, and result ordering.

## Expected Results Summary

- **Perfect matches**: Score = 1.0 for identical artist lists
- **High similarity**: Score > 0.8 for very similar preferences
- **Moderate similarity**: Score 0.4-0.7 for some shared artists with position considerations
- **Low similarity**: Score < 0.4 for minimal overlap or poor positional alignment
- **No overlap**: No matches returned (engine filters out zero-overlap cases)
- **Result ordering**: Matches sorted by score (descending), then by overlap (descending)

These tests ensure that the music matching algorithm remains consistent and produces scientifically sound PWO-based similarity calculations.