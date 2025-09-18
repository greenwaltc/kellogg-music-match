# Music Matching Behavioral Tests

This directory contains comprehensive behavioral tests for the music matching system using the Ginkgo testing framework.

## Test Coverage

The behavioral tests verify the following aspects of the music matching algorithm:

### Core Matching Scenarios
- **Identical Preferences**: Users with exactly the same artist preferences should get maximum similarity scores
- **Subset Relationships**: Users whose preferences are subsets of others should get good but not perfect scores  
- **No Overlap**: Users with completely different preferences should get no matches
- **Partial Overlap**: Users with some shared artists should get moderate similarity scores based on Jaccard similarity

### Edge Cases
- Empty artist lists
- Single artist preferences
- Caller exclusion from results
- Whitespace and case normalization

### Algorithm Validation
- **Jaccard Similarity Calculation**: Validates that the engine correctly calculates Jaccard coefficients
- **Database Function Alignment**: Tests scenarios that correspond to the PostgreSQL `spearman_distance` function outcomes:
  - Distance = 0 (identical arrays) → High similarity scores
  - Distance = 0.7 (subset relationships) → Moderate similarity scores  
  - Distance = 2.0 (no overlap) → No matches returned

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
2. **Algorithm Validation Against Database Distance Function**: Validates alignment with PostgreSQL similarity calculations

Each test creates specific user scenarios and validates that the matching engine produces expected results for similarity scores, overlap counts, and result ordering.

## Expected Results Summary

- **Perfect matches**: Score ≥ 0.9, all artists overlap
- **Subset matches**: Score = 0.5 for 50% Jaccard similarity
- **Partial overlap**: Score = 0.33 for 1/3 Jaccard similarity  
- **No overlap**: No matches returned (engine filters out zero-overlap cases)
- **Result ordering**: Matches sorted by score (descending), then by overlap (descending)

These tests ensure that the music matching algorithm remains consistent and produces scientifically sound similarity calculations.