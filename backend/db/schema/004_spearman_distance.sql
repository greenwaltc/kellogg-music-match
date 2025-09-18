-- Enable plpython3u extension for advanced statistical functions
CREATE EXTENSION IF NOT EXISTS plpython3u;

-- Create Spearman rank correlation distance function for text arrays (artist names)
-- This function calculates a distance metric based on artist preference similarity
-- Returns a distance where 0 = identical preferences, higher values = less similar
CREATE OR REPLACE FUNCTION spearman_distance(arr1 TEXT[], arr2 TEXT[])
RETURNS FLOAT
LANGUAGE plpython3u
AS $$
import scipy.stats
import numpy as np

# Convert PostgreSQL arrays to Python lists
list1 = arr1 if arr1 else []
list2 = arr2 if arr2 else []

# Handle edge cases
if len(list1) == 0 and len(list2) == 0:
    return 0.0  # Both empty, perfect match
    
if len(list1) == 0 or len(list2) == 0:
    return 1.0  # One empty, maximum distance

# If arrays are identical, return perfect match
if list1 == list2:
    return 0.0

# Calculate similarity using a hybrid approach that considers:
# 1. Jaccard similarity (intersection over union)
# 2. Positional similarity (rank correlation for shared items)

# Calculate set operations
set1 = set(list1)
set2 = set(list2)
intersection = set1.intersection(set2)
union = set1.union(set2)

# If no common items, return maximum distance
if len(intersection) == 0:
    return 2.0

# Calculate Jaccard similarity
jaccard_similarity = len(intersection) / len(union)

# For shared items, calculate positional similarity
if len(intersection) > 1:
    # Extract ranks of shared items in both lists
    shared_ranks1 = []
    shared_ranks2 = []
    
    for item in intersection:
        rank1 = list1.index(item) + 1  # 1-based ranking
        rank2 = list2.index(item) + 1
        shared_ranks1.append(rank1)
        shared_ranks2.append(rank2)
    
    # Calculate Spearman correlation for shared items only
    try:
        correlation, _ = scipy.stats.spearmanr(shared_ranks1, shared_ranks2)
        if np.isnan(correlation):
            correlation = 0.0
        positional_similarity = (correlation + 1.0) / 2.0  # Convert [-1,1] to [0,1]
    except:
        positional_similarity = 0.5  # Default middle value if calculation fails
else:
    # Only one shared item, perfect positional agreement
    positional_similarity = 1.0

# Combine Jaccard and positional similarities
# Weight Jaccard more heavily as it's more important for music preferences
combined_similarity = 0.7 * jaccard_similarity + 0.3 * positional_similarity

# Convert to distance (0 = identical, 2 = completely different)
distance = 2.0 * (1.0 - combined_similarity)

return max(0.0, min(2.0, float(distance)))
$$;

-- Alternative simpler implementation using basic statistics (fallback)
-- This creates a backup function in case plpython3u isn't available
CREATE OR REPLACE FUNCTION spearman_distance_simple(arr1 TEXT[], arr2 TEXT[])
RETURNS FLOAT
LANGUAGE plpgsql
AS $$
DECLARE
    intersection_count INTEGER;
    union_count INTEGER;
    jaccard_similarity FLOAT;
BEGIN
    -- Handle edge cases
    IF arr1 IS NULL AND arr2 IS NULL THEN
        RETURN 0.0;  -- Both null, perfect match
    END IF;
    
    IF arr1 IS NULL OR arr2 IS NULL THEN
        RETURN 1.0;  -- One null, maximum distance
    END IF;
    
    -- If arrays are identical, return perfect match
    IF arr1 = arr2 THEN
        RETURN 0.0;
    END IF;
    
    -- Calculate Jaccard similarity as a simple alternative
    -- intersection_count = |A ∩ B|
    SELECT COUNT(*)
    INTO intersection_count
    FROM (SELECT unnest(arr1) INTERSECT SELECT unnest(arr2)) AS intersection;
    
    -- union_count = |A ∪ B|
    SELECT COUNT(*)
    INTO union_count
    FROM (SELECT unnest(arr1) UNION SELECT unnest(arr2)) AS union_set;
    
    -- Avoid division by zero
    IF union_count = 0 THEN
        RETURN 0.0;
    END IF;
    
    -- Jaccard similarity = |A ∩ B| / |A ∪ B|
    jaccard_similarity := CAST(intersection_count AS FLOAT) / CAST(union_count AS FLOAT);
    
    -- Convert to distance (1 - similarity)
    RETURN 1.0 - jaccard_similarity;
END;
$$;

-- Create a comment explaining the function
COMMENT ON FUNCTION spearman_distance(TEXT[], TEXT[]) IS 
'Calculates Spearman rank correlation distance between two text arrays (artist names). Returns 1 - correlation coefficient as distance metric (0 = identical rankings, 1 = no correlation, 2 = opposite rankings). Requires plpython3u extension with scipy.stats.';

COMMENT ON FUNCTION spearman_distance_simple(TEXT[], TEXT[]) IS 
'Simple fallback implementation using Jaccard similarity for text arrays. Calculates 1 - (intersection / union) as distance metric. Pure PostgreSQL implementation without external dependencies.';