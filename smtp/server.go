package smtp

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"
	"github.com/mailvex/mailvex/core"
	"github.com/mailvex/mailvex/core/crypto"
	"github.com/mailvex/mailvex/core/domain"
	"github.com/mailvex/mailvex/core/repository"
)

var (
	ErrAuthFailed     = &gosmtp.SMTPError{Code: 535, EnhancedCode: gosmtp.EnhancedCode{5, 7, 8}, Message: "Authentication credentials invalid"}
	ErrRelayingDenied = &gosmtp.SMTPError{Code: 550, EnhancedCode: gosmtp.EnhancedCode{5, 7, 1}, Message: "Relaying denied: authentication required"}
	ErrSenderDenied   = &gosmtp.SMTPError{Code: 550, EnhancedCode: gosmtp.EnhancedCode{5, 7, 1}, Message: "Sender address not authorized for this credential"}
	ErrIPBlocked      = &gosmtp.SMTPError{Code: 421, EnhancedCode: gosmtp.EnhancedCode{4, 7, 0}, Message: "Too many authentication failures, IP temporarily blocked"}
)

type Backend struct {
	repo       repository.Repository
	ingest     *core.IngestionEngine
	mu         sync.Mutex
	authFails  map[string]int       // ip -> fail count
	blockedIPs map[string]time.Time // ip -> unblock time
}

func NewBackend(repo repository.Repository, ingest *core.IngestionEngine) *Backend {
	return &Backend{
		repo:       repo,
		ingest:     ingest,
		authFails:  make(map[string]int),
		blockedIPs: make(map[string]time.Time),
	}
}

func (b *Backend) NewSession(c *gosmtp.Conn) (gosmtp.Session, error) {
	clientIP := ""
	if addr, ok := c.Conn().RemoteAddr().(*net.TCPAddr); ok {
		clientIP = addr.IP.String()
	}

	b.mu.Lock()
	if unblockTime, blocked := b.blockedIPs[clientIP]; blocked {
		if time.Now().Before(unblockTime) {
			b.mu.Unlock()
			return nil, ErrIPBlocked
		}
		delete(b.blockedIPs, clientIP)
		delete(b.authFails, clientIP)
	}
	b.mu.Unlock()

	return &Session{
		backend:  b,
		clientIP: clientIP,
	}, nil
}

type Session struct {
	backend       *Backend
	clientIP      string
	authenticated bool
	cred          *domain.SMTPCredential
	device        *domain.Device
	tenantID      string
	from          string
	to            []string
}

func (s *Session) AuthMechanisms() []string {
	return []string{sasl.Plain}
}

func (s *Session) Auth(mech string) (sasl.Server, error) {
	if mech != sasl.Plain {
		return nil, errors.New("unsupported authentication mechanism")
	}

	return sasl.NewPlainServer(func(identity, username, password string) error {
		ctx := context.Background()
		cred, err := s.backend.repo.GetSMTPCredentialByUsername(ctx, username)
		if err != nil {
			s.recordAuthFailure()
			return ErrAuthFailed
		}

		match, err := crypto.VerifyPassword(password, cred.PasswordHash)
		if err != nil || !match {
			s.recordAuthFailure()
			return ErrAuthFailed
		}

		s.authenticated = true
		s.cred = cred
		s.tenantID = cred.TenantID
		return nil
	}), nil
}

func (s *Session) recordAuthFailure() {
	s.backend.mu.Lock()
	defer s.backend.mu.Unlock()

	s.backend.authFails[s.clientIP]++
	if s.backend.authFails[s.clientIP] >= 5 {
		s.backend.blockedIPs[s.clientIP] = time.Now().Add(15 * time.Minute)
	}
}

// Mail handles MAIL FROM command
func (s *Session) Mail(from string, opts *gosmtp.MailOptions) error {
	ctx := context.Background()

	// If not authenticated, check for Trusted Network IP-authenticated device (§28.1)
	if !s.authenticated {
		tenants, _ := s.backend.repo.ListOrganizations(ctx)
		// Look for a registered device matching client IP in any tenant
		for _, org := range tenants {
			tList, _ := s.backend.repo.ListTenantsByOrg(ctx, org.ID)
			for _, t := range tList {
				inTrusted, _ := s.backend.repo.IsIPInTrustedNetwork(ctx, t.ID, s.clientIP)
				if inTrusted {
					dev, err := s.backend.repo.GetDeviceByIP(ctx, t.ID, s.clientIP)
					if err == nil && dev != nil {
						s.device = dev
						s.tenantID = t.ID
						s.authenticated = true
						break
					}
				}
			}
			if s.authenticated {
				break
			}
		}
	}

	// If device is authenticated, verify allowed From addresses
	if s.device != nil && len(s.device.AllowedFrom) > 0 {
		allowed := false
		for _, a := range s.device.AllowedFrom {
			if strings.EqualFold(a, from) {
				allowed = true
				break
			}
		}
		if !allowed {
			return ErrSenderDenied
		}
	}

	// If still unauthenticated, relaying will be checked on RCPT TO
	s.from = from
	s.to = nil
	return nil
}

// Rcpt handles RCPT TO command and enforces open relay prevention (§4)
func (s *Session) Rcpt(to string, opts *gosmtp.RcptOptions) error {
	ctx := context.Background()

	// If session is authenticated (or verified IoT device):
	if s.authenticated {
		s.to = append(s.to, to)
		return nil
	}

	// Open Relay Prevention (§4):
	// Unauthenticated sessions may ONLY deliver to verified inbound domains hosted on Mailvex
	parts := strings.Split(to, "@")
	if len(parts) != 2 {
		return ErrRelayingDenied
	}
	rcptDomain := strings.ToLower(parts[1])

	dom, err := s.backend.repo.GetDomainByName(ctx, rcptDomain)
	if err != nil || (dom.Status != "verified" && dom.Status != "degraded") {
		// External destination without auth -> RELAYING DENIED
		return ErrRelayingDenied
	}

	// Allowed inbound destination on Mailvex
	s.tenantID = dom.TenantID
	s.to = append(s.to, to)
	return nil
}

// Data handles DATA command: ingests message into Core pipeline
func (s *Session) Data(r io.Reader) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	ctx := context.Background()
	var credID *string
	if s.cred != nil {
		credID = &s.cred.ID
	}

	_, err = s.backend.ingest.IngestRawMIME(ctx, s.tenantID, s.from, s.to, raw, credID)
	if err != nil {
		if errors.Is(err, core.ErrDomainNotVerified) {
			return ErrSenderDenied
		}
		return &gosmtp.SMTPError{Code: 451, EnhancedCode: gosmtp.EnhancedCode{4, 3, 0}, Message: err.Error()}
	}

	return nil
}

func (s *Session) Reset() {
	s.from = ""
	s.to = nil
}

func (s *Session) Logout() error {
	return nil
}

// Server wraps gosmtp.Server
type Server struct {
	server *gosmtp.Server
}

func NewServer(addr, hostname string, backend *Backend) *Server {
	s := gosmtp.NewServer(backend)
	s.Addr = addr
	s.Domain = hostname
	s.ReadTimeout = 30 * time.Second
	s.WriteTimeout = 30 * time.Second
	s.MaxMessageBytes = 25 * 1024 * 1024 // 25 MB default limit
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true // Permitted for testing & LAN IoT submission

	return &Server{server: s}
}

func (srv *Server) ListenAndServe() error {
	return srv.server.ListenAndServe()
}

func (srv *Server) Close() error {
	return srv.server.Close()
}
