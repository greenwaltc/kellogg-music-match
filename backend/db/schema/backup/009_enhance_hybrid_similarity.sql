-- Enhanced hybrid similarity function with size penalty for variable-length lists
-- This migration updates the spearman_distance function to handle lists of very different sizes better

-- Drop the existing function and recreate with enhancements
DROP FUNCTION IF EXISTS spearman_distance(TEXT[], TEXT[]);

-- Create enhanced Spearman rank correlation distance function for text arrays (artist names)
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
# 3. Size penalty for very different list lengths

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

# Calculate size penalty for very different list lengths
len1, len2 = len(list1), len(list2)
size_ratio = min(len1, len2) / max(len1, len2) if max(len1, len2) > 0 else 1.0
# Size penalty ranges from 0 (very different sizes) to 1 (same size)
# Apply penalty more strongly for extreme size differences
if size_ratio < 0.5:  # One list is more than 2x the other
    size_penalty = size_ratio * 0.5  # Stronger penalty
else:
    size_penalty = 0.5 + (size_ratio - 0.5)  # Gentle penalty

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

# Combine Jaccard, positional similarities, and size penalty
# Adjust weights to account for size penalty
# For music preferences: Jaccard (overlap) is most important, 
# position matters less, and size differences should be penalized
combined_similarity = (0.6 * jaccard_similarity + 0.2 * positional_similarity) * size_penalty

# Convert to distance (0 = identical, 2 = completely different)
distance = 2.0 * (1.0 - combined_similarity)

return max(0.0, min(2.0, float(distance)))
$$;

-- Update the comment explaining the enhanced function
COMMENT ON FUNCTION spearman_distance(TEXT[], TEXT[]) IS 
'Enhanced hybrid similarity function that calculates distance between two text arrays (artist names). Combines Jaccard similarity (60%), positional correlation (20%), and size penalty for variable-length lists. Returns 0 = identical rankings, 2 = completely different. Requires plpython3u extension with scipy.stats.';