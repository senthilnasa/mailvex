package crypto

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// ID Prefix constants
const (
	PrefixOrg          = "org_"
	PrefixTenant       = "tenant_"
	PrefixUser         = "user_"
	PrefixDomain       = "dom_"
	PrefixMessage      = "msg_"
	PrefixRecipient    = "rcpt_"
	PrefixAttempt      = "att_"
	PrefixEvent        = "evt_"
	PrefixAPIKey       = "key_"
	PrefixSMTPCred     = "smtp_"
	PrefixWebhook      = "wh_"
	PrefixWebhookDeliv = "whd_"
	PrefixIP           = "ip_"
	PrefixPool         = "pool_"
	PrefixStream       = "stream_"
	PrefixSuppression       = "sup_"
	PrefixSuppressionImport = "supimp_"
	PrefixPrivacyRequest    = "priv_"
	PrefixRetentionPolicy   = "retpol_"
	PrefixRequest           = "req_"
	PrefixAudit             = "aud_"
	PrefixMembership        = "mem_"
	PrefixTemplate          = "tmpl_"
	PrefixTemplateVer       = "tmplver_"
	PrefixTrackingLink      = "tl_"
	PrefixTrackingEvent     = "te_"
	PrefixInboundRoute      = "inr_"
	PrefixInboundMessage    = "inm_"
)

// NewULID generates a new ULID string with the given prefix
func NewULID(prefix string) string {
	entropy := ulid.Monotonic(rand.Reader, 0)
	id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
	return prefix + strings.ToLower(id.String())
}

// ParseULID validates and extracts the raw ULID component from a prefixed string
func ParseULID(prefixedID, expectedPrefix string) (ulid.ULID, error) {
	if !strings.HasPrefix(prefixedID, expectedPrefix) {
		return ulid.ULID{}, fmt.Errorf("id %s does not have expected prefix %s", prefixedID, expectedPrefix)
	}
	raw := strings.ToUpper(strings.TrimPrefix(prefixedID, expectedPrefix))
	return ulid.Parse(raw)
}
