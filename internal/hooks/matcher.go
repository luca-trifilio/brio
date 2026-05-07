package hooks

import (
	"regexp"

	"github.com/luca-trifilio/brio/internal/config"
	"github.com/luca-trifilio/brio/internal/httpx"
	"github.com/luca-trifilio/brio/internal/theme"
)

// Match returns the first hook whose trigger conditions are satisfied by the
// given HTTP response and environment tier, or nil if none match.
//
// Matching rules (all must pass):
//  1. response StatusCode is in hook.trigger.status
//  2. if hook.trigger.body is set, it must match as a regex against response body
//  3. if hook.trigger.tier is set, it must equal the current env tier
func Match(hooks []config.Hook, resp httpx.Response, tier theme.EnvTier) *config.Hook {
	for i := range hooks {
		h := &hooks[i]

		// 1. status code
		if !statusMatches(h.Trigger.Status, resp.StatusCode) {
			continue
		}

		// 2. optional body regex
		if h.Trigger.Body != "" {
			re, err := regexp.Compile(h.Trigger.Body)
			if err != nil || !re.Match(resp.Body) {
				continue
			}
		}

		// 3. optional tier
		if h.Trigger.Tier != "" && !tierMatches(h.Trigger.Tier, tier) {
			continue
		}

		return h
	}
	return nil
}

// statusMatches reports whether code appears in the list.
func statusMatches(list []int, code int) bool {
	for _, s := range list {
		if s == code {
			return true
		}
	}
	return false
}

// tierMatches reports whether the string tier name maps to the given EnvTier.
func tierMatches(name string, tier theme.EnvTier) bool {
	switch name {
	case "safe":
		return tier == theme.TierSafe
	case "caution":
		return tier == theme.TierCaution
	case "danger":
		return tier == theme.TierDanger
	default:
		return false
	}
}
