package dns

import (
	"fmt"
	"os"
	"strings"
)

// SPFHost returns the hostname customer domains should reference with
// `include:` in their SPF record.
//
// It is MVX_SPF_HOST when set, falling back to platformHost. A separate
// setting is needed because the platform host is often a CNAME (to a proxy
// or load balancer), and DNS forbids a TXT record beside a CNAME at the same
// name. Including a name with no SPF record yields SPF PERMERROR at receivers,
// so operators publish the sending IPs at a dedicated name such as
// `_spf.mail.example.com` and point MVX_SPF_HOST at it.
func SPFHost(platformHost string) string {
	if h := strings.TrimSuffix(strings.TrimSpace(os.Getenv("MVX_SPF_HOST")), "."); h != "" {
		return h
	}
	return platformHost
}

// ExpectedSPF returns the SPF record customer domains should publish.
func ExpectedSPF(spfHost string) string {
	return fmt.Sprintf("v=spf1 include:%s -all", spfHost)
}

// spfIncludes reports whether an SPF record contains an include: mechanism
// (with any qualifier) for host.
func spfIncludes(record, host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for term := range strings.FieldsSeq(strings.ToLower(record)) {
		term = strings.TrimLeft(term, "+-~?")
		if v, ok := strings.CutPrefix(term, "include:"); ok && strings.TrimSuffix(v, ".") == host {
			return true
		}
	}
	return false
}
