CREATE EXTENSION IF NOT EXISTS plpython3u;

CREATE OR REPLACE FUNCTION spearman_distance(list1 TEXT[], list2 TEXT[])
RETURNS FLOAT AS $$
    # 1. Find the union of all preferences
    all_items = set(list1) | set(list2)
    n = len(all_items)

    # If lists are too small, correlation is undefined. Return max distance.
    if n <= 1:
        return 2.0

    # 2. Create rank dictionaries for each list
    ranks1 = {item: i + 1 for i, item in enumerate(list1)}
    ranks2 = {item: i + 1 for i, item in enumerate(list2)}

    # 3. Calculate sum of squared differences (d^2)
    sum_sq_diff = 0
    for item in all_items:
        # Assign a penalty rank (n + 1) if an item is missing
        rank1 = ranks1.get(item, n + 1)
        rank2 = ranks2.get(item, n + 1)
        diff = rank1 - rank2
        sum_sq_diff += diff ** 2

    # 4. Calculate Spearman's rank correlation coefficient (rho)
    rho = 1 - (6 * sum_sq_diff) / (n * (n**2 - 1))

    # 5. Convert correlation to distance
    distance = 1 - rho
    return distance

$$ LANGUAGE plpython3u;