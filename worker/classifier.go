package worker

import (
	"regexp"
	"strings"

	"github.com/senthilnasa/mailvex/core/domain"
)

type BounceClassification struct {
	Classification string // hard_bounce, soft_bounce, policy_rejection, spam_rejection, auth_rejection, rate_limited, mailbox_unavailable, domain_unavailable, unknown
	Action         string // suppress_immediate, retry, dead_letter, alert
	Reason         string
}

type Classifier struct {
	rules []*domain.BounceRule
}

func NewClassifier(rules []*domain.BounceRule) *Classifier {
	return &Classifier{
		rules: rules,
	}
}

// Classify categorizes SMTP response codes and error messages (§16)
func (c *Classifier) Classify(smtpCode, enhancedCode, responseText string) BounceClassification {
	text := strings.ToLower(responseText)
	enh := strings.TrimSpace(enhancedCode)
	code := strings.TrimSpace(smtpCode)

	// Check dynamic bounce rules first
	for _, r := range c.rules {
		if r.EnhancedCode != nil && *r.EnhancedCode != "" && enh == *r.EnhancedCode {
			return BounceClassification{
				Classification: r.Classification,
				Action:         r.Action,
				Reason:         responseText,
			}
		}
		if r.Pattern != "" {
			if matched, _ := regexp.MatchString("(?i)"+r.Pattern, text); matched {
				return BounceClassification{
					Classification: r.Classification,
					Action:         r.Action,
					Reason:         responseText,
				}
			}
		}
	}

	// 1. Rate limited checks
	if enh == "4.7.28" || strings.HasPrefix(enh, "4.7.") || strings.Contains(text, "rate limit") || strings.Contains(text, "too many connections") || strings.Contains(text, "throttled") {
		return BounceClassification{
			Classification: "rate_limited",
			Action:         "retry",
			Reason:         responseText,
		}
	}

	// 2. Mailbox full / quota exceeded (soft bounce)
	if enh == "4.2.2" || strings.Contains(text, "mailbox full") || strings.Contains(text, "quota exceeded") || strings.Contains(text, "over quota") {
		return BounceClassification{
			Classification: "soft_bounce",
			Action:         "retry",
			Reason:         responseText,
		}
	}

	// 3. Mailbox unavailable
	if enh == "4.2.1" || strings.Contains(text, "mailbox unavailable") {
		return BounceClassification{
			Classification: "mailbox_unavailable",
			Action:         "retry",
			Reason:         responseText,
		}
	}

	// 4. Hard bounce checks (user unknown, bad domain)
	if enh == "5.1.1" || enh == "5.1.2" || enh == "5.1.10" ||
		strings.Contains(text, "user unknown") ||
		strings.Contains(text, "mailbox not found") ||
		strings.Contains(text, "no such user") ||
		strings.Contains(text, "recipient address rejected") ||
		strings.Contains(text, "invalid recipient") ||
		strings.Contains(text, "unrouteable address") ||
		strings.Contains(text, "domain not found") {
		return BounceClassification{
			Classification: "hard_bounce",
			Action:         "suppress_immediate",
			Reason:         responseText,
		}
	}

	// 5. Spam rejection
	if strings.Contains(text, "spam") || strings.Contains(text, "blocked using") || strings.Contains(text, "reputation") || strings.Contains(text, "blacklisted") {
		return BounceClassification{
			Classification: "spam_rejection",
			Action:         "dead_letter",
			Reason:         responseText,
		}
	}

	// 6. Authentication rejection (DKIM/SPF/DMARC)
	if enh == "5.7.26" || strings.Contains(text, "dkim") || strings.Contains(text, "spf") || strings.Contains(text, "dmarc") {
		return BounceClassification{
			Classification: "auth_rejection",
			Action:         "dead_letter",
			Reason:         responseText,
		}
	}

	// 7. Policy rejection
	if enh == "5.7.1" || strings.Contains(text, "not authorized") || strings.Contains(text, "access denied") {
		return BounceClassification{
			Classification: "policy_rejection",
			Action:         "dead_letter",
			Reason:         responseText,
		}
	}

	// 8. General 5xx -> permanent failure (hard bounce fallback)
	if strings.HasPrefix(code, "5") || strings.HasPrefix(enh, "5.") {
		return BounceClassification{
			Classification: "hard_bounce",
			Action:         "suppress_immediate",
			Reason:         responseText,
		}
	}

	// 9. General 4xx -> transient failure (soft bounce fallback)
	if strings.HasPrefix(code, "4") || strings.HasPrefix(enh, "4.") {
		return BounceClassification{
			Classification: "soft_bounce",
			Action:         "retry",
			Reason:         responseText,
		}
	}

	return BounceClassification{
		Classification: "unknown",
		Action:         "retry",
		Reason:         responseText,
	}
}
