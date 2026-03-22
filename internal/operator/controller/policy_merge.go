package controller

import (
	"strconv"

	v1alpha1 "github.com/szaher/agentspec/internal/api/v1alpha1"
)

// MergePolicies merges multiple PolicySpecFields using a most-restrictive-wins strategy.
// - Cost budgets: take the lowest maxDailyCost.
// - Allowed models: intersection (only models present in all policies).
// - Denied models: union (any model denied by any policy is denied).
// - Rate limits: take the lowest values.
// - Content filters: union (all filters from all policies).
// - Tool restrictions: intersection of allowed, union of denied.
func MergePolicies(policies []v1alpha1.PolicySpecFields) v1alpha1.PolicySpecFields {
	if len(policies) == 0 {
		return v1alpha1.PolicySpecFields{}
	}
	if len(policies) == 1 {
		return policies[0]
	}

	result := v1alpha1.PolicySpecFields{}

	// Merge cost budgets: take the lowest maxDailyCost.
	result.CostBudget = mergeCostBudgets(policies)

	// Merge allowed models: intersection.
	result.AllowedModels = mergeAllowedModels(policies)

	// Merge denied models: union.
	result.DeniedModels = mergeDeniedModels(policies)

	// Merge rate limits: take the lowest values.
	result.RateLimits = mergeRateLimits(policies)

	// Merge content filters: union.
	result.ContentFilters = mergeContentFilters(policies)

	// Merge tool restrictions: intersection of allowed, union of denied.
	result.ToolRestrictions = mergeToolRestrictions(policies)

	return result
}

// mergeCostBudgets takes the lowest maxDailyCost across all policies.
func mergeCostBudgets(policies []v1alpha1.PolicySpecFields) *v1alpha1.CostBudget {
	var lowestCost float64
	var lowestCurrency string
	found := false

	for _, p := range policies {
		if p.CostBudget == nil {
			continue
		}
		val, err := strconv.ParseFloat(p.CostBudget.MaxDailyCost, 64)
		if err != nil {
			continue
		}
		if !found || val < lowestCost {
			lowestCost = val
			lowestCurrency = p.CostBudget.Currency
			found = true
		}
	}

	if !found {
		return nil
	}
	return &v1alpha1.CostBudget{
		MaxDailyCost: strconv.FormatFloat(lowestCost, 'f', -1, 64),
		Currency:     lowestCurrency,
	}
}

// mergeAllowedModels computes the intersection of allowed models.
func mergeAllowedModels(policies []v1alpha1.PolicySpecFields) []string {
	// Collect only policies that specify allowed models.
	var sets []map[string]bool
	for _, p := range policies {
		if len(p.AllowedModels) > 0 {
			s := make(map[string]bool, len(p.AllowedModels))
			for _, m := range p.AllowedModels {
				s[m] = true
			}
			sets = append(sets, s)
		}
	}

	if len(sets) == 0 {
		return nil
	}

	// Start with the first set and intersect with the rest.
	intersection := sets[0]
	for _, s := range sets[1:] {
		for k := range intersection {
			if !s[k] {
				delete(intersection, k)
			}
		}
	}

	result := make([]string, 0, len(intersection))
	for k := range intersection {
		result = append(result, k)
	}
	return result
}

// mergeDeniedModels computes the union of denied models.
func mergeDeniedModels(policies []v1alpha1.PolicySpecFields) []string {
	seen := make(map[string]bool)
	for _, p := range policies {
		for _, m := range p.DeniedModels {
			seen[m] = true
		}
	}
	if len(seen) == 0 {
		return nil
	}
	result := make([]string, 0, len(seen))
	for k := range seen {
		result = append(result, k)
	}
	return result
}

// mergeRateLimits takes the lowest rate limit values.
func mergeRateLimits(policies []v1alpha1.PolicySpecFields) *v1alpha1.RateLimits {
	var minRPM int32
	var minTPM int64
	rpmSet := false
	tpmSet := false

	for _, p := range policies {
		if p.RateLimits == nil {
			continue
		}
		if p.RateLimits.RequestsPerMinute > 0 {
			if !rpmSet || p.RateLimits.RequestsPerMinute < minRPM {
				minRPM = p.RateLimits.RequestsPerMinute
				rpmSet = true
			}
		}
		if p.RateLimits.TokensPerMinute > 0 {
			if !tpmSet || p.RateLimits.TokensPerMinute < minTPM {
				minTPM = p.RateLimits.TokensPerMinute
				tpmSet = true
			}
		}
	}

	if !rpmSet && !tpmSet {
		return nil
	}

	rl := &v1alpha1.RateLimits{}
	if rpmSet {
		rl.RequestsPerMinute = minRPM
	}
	if tpmSet {
		rl.TokensPerMinute = minTPM
	}
	return rl
}

// mergeContentFilters computes the union of all content filters.
func mergeContentFilters(policies []v1alpha1.PolicySpecFields) []v1alpha1.ContentFilter {
	type filterKey struct {
		Type    string
		Pattern string
	}
	seen := make(map[filterKey]bool)
	var result []v1alpha1.ContentFilter

	for _, p := range policies {
		for _, f := range p.ContentFilters {
			k := filterKey{Type: f.Type, Pattern: f.Pattern}
			if !seen[k] {
				seen[k] = true
				result = append(result, f)
			}
		}
	}
	return result
}

// mergeToolRestrictions computes intersection of allowed tools, union of denied tools.
func mergeToolRestrictions(policies []v1alpha1.PolicySpecFields) *v1alpha1.ToolRestrictions {
	var allowedSets []map[string]bool
	deniedSeen := make(map[string]bool)
	hasRestrictions := false

	for _, p := range policies {
		if p.ToolRestrictions == nil {
			continue
		}
		hasRestrictions = true

		if len(p.ToolRestrictions.AllowedTools) > 0 {
			s := make(map[string]bool, len(p.ToolRestrictions.AllowedTools))
			for _, t := range p.ToolRestrictions.AllowedTools {
				s[t] = true
			}
			allowedSets = append(allowedSets, s)
		}

		for _, t := range p.ToolRestrictions.DeniedTools {
			deniedSeen[t] = true
		}
	}

	if !hasRestrictions {
		return nil
	}

	tr := &v1alpha1.ToolRestrictions{}

	// Intersection of allowed tools.
	if len(allowedSets) > 0 {
		intersection := make(map[string]bool)
		for k, v := range allowedSets[0] {
			intersection[k] = v
		}
		for _, s := range allowedSets[1:] {
			for k := range intersection {
				if !s[k] {
					delete(intersection, k)
				}
			}
		}
		allowed := make([]string, 0, len(intersection))
		for k := range intersection {
			allowed = append(allowed, k)
		}
		tr.AllowedTools = allowed
	}

	// Union of denied tools.
	if len(deniedSeen) > 0 {
		denied := make([]string, 0, len(deniedSeen))
		for k := range deniedSeen {
			denied = append(denied, k)
		}
		tr.DeniedTools = denied
	}

	return tr
}
