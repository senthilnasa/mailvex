package worker

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/mailvex/mailvex/core/domain"
)

type DeliveryRequest struct {
	RecipientID   string
	RecipientAddr string
	ReturnPath    string // VERP return-path
	MIMEData      []byte
	EHLOHostname  string
	SourceIP      string
	TLSPolicy     string // "opportunistic", "required", "none"
	// Destination info
	RemoteMX   string
	RemotePort int // usually 25, configurable for testing
	// For Relay transport
	Relay         *domain.Relay
	RelayAuthPass string // decrypted password
}

type DeliveryResponse struct {
	Success          bool
	SMTPCode         string
	EnhancedCode     string
	ResponseText     string
	RemoteIP         string
	RemoteMX         string
	TLSVersion       string
	TLSCipher        string
	TLSPolicyApplied string
	Duration         time.Duration
}

type Transport interface {
	Deliver(ctx context.Context, req *DeliveryRequest) (*DeliveryResponse, error)
}

type DirectMXTransport struct {
	DialTimeout time.Duration
	IOTimeout   time.Duration
}

func NewDirectMXTransport() *DirectMXTransport {
	return &DirectMXTransport{
		DialTimeout: 30 * time.Second,
		IOTimeout:   60 * time.Second,
	}
}

func (t *DirectMXTransport) Deliver(ctx context.Context, req *DeliveryRequest) (*DeliveryResponse, error) {
	start := time.Now()
	port := req.RemotePort
	if port <= 0 {
		port = 25
	}
	targetAddr := fmt.Sprintf("%s:%d", req.RemoteMX, port)

	var dialer net.Dialer
	dialer.Timeout = t.DialTimeout
	if req.SourceIP != "" && req.SourceIP != "127.0.0.1" && req.SourceIP != "localhost" {
		if localIP := net.ParseIP(req.SourceIP); localIP != nil {
			dialer.LocalAddr = &net.TCPAddr{IP: localIP}
		}
	}

	conn, err := dialer.DialContext(ctx, "tcp", targetAddr)
	if err != nil {
		return &DeliveryResponse{
			Success:      false,
			SMTPCode:     "451",
			EnhancedCode: "4.4.1",
			ResponseText: fmt.Sprintf("connection failed to %s: %v", targetAddr, err),
			RemoteMX:     req.RemoteMX,
			Duration:     time.Since(start),
		}, nil
	}
	defer conn.Close()

	remoteIP := ""
	if tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		remoteIP = tcpAddr.IP.String()
	}

	_ = conn.SetDeadline(time.Now().Add(t.IOTimeout))

	client, err := smtp.NewClient(conn, req.RemoteMX)
	if err != nil {
		code, enh := parseProtoError(err)
		return &DeliveryResponse{
			Success:      false,
			SMTPCode:     code,
			EnhancedCode: enh,
			ResponseText: fmt.Sprintf("greeting error: %v", err),
			RemoteIP:     remoteIP,
			RemoteMX:     req.RemoteMX,
			Duration:     time.Since(start),
		}, nil
	}
	defer client.Close()

	// EHLO
	ehloHost := req.EHLOHostname
	if ehloHost == "" {
		ehloHost = "mailvex.local"
	}
	if err := client.Hello(ehloHost); err != nil {
		code, enh := parseProtoError(err)
		return &DeliveryResponse{
			Success:      false,
			SMTPCode:     code,
			EnhancedCode: enh,
			ResponseText: fmt.Sprintf("EHLO rejected: %v", err),
			RemoteIP:     remoteIP,
			RemoteMX:     req.RemoteMX,
			Duration:     time.Since(start),
		}, nil
	}

	// STARTTLS
	tlsPolicy := req.TLSPolicy
	if tlsPolicy == "" {
		tlsPolicy = "opportunistic"
	}
	var tlsVersion, tlsCipher, tlsPolicyApplied string

	supportsTLS, _ := client.Extension("STARTTLS")
	if supportsTLS && (tlsPolicy == "opportunistic" || tlsPolicy == "required") {
		tlsConf := &tls.Config{
			ServerName:         req.RemoteMX,
			InsecureSkipVerify: (tlsPolicy == "opportunistic"),
		}
		if err := client.StartTLS(tlsConf); err == nil {
			if state, ok := client.TLSConnectionState(); ok {
				tlsVersion = tlsVersionName(state.Version)
				tlsCipher = tls.CipherSuiteName(state.CipherSuite)
				tlsPolicyApplied = tlsPolicy
			}
		}
	} else if tlsPolicy == "required" && !supportsTLS {
		return &DeliveryResponse{
			Success:          false,
			SMTPCode:         "550",
			EnhancedCode:     "5.7.10",
			ResponseText:     "TLS required by policy but STARTTLS not supported by remote MX",
			RemoteIP:         remoteIP,
			RemoteMX:         req.RemoteMX,
			TLSPolicyApplied: "required_failed",
			Duration:         time.Since(start),
		}, nil
	}

	// MAIL FROM
	if err := client.Mail(req.ReturnPath); err != nil {
		code, enh := parseProtoError(err)
		return &DeliveryResponse{
			Success:          false,
			SMTPCode:         code,
			EnhancedCode:     enh,
			ResponseText:     err.Error(),
			RemoteIP:         remoteIP,
			RemoteMX:         req.RemoteMX,
			TLSVersion:       tlsVersion,
			TLSCipher:        tlsCipher,
			TLSPolicyApplied: tlsPolicyApplied,
			Duration:         time.Since(start),
		}, nil
	}

	// RCPT TO
	if err := client.Rcpt(req.RecipientAddr); err != nil {
		code, enh := parseProtoError(err)
		return &DeliveryResponse{
			Success:          false,
			SMTPCode:         code,
			EnhancedCode:     enh,
			ResponseText:     err.Error(),
			RemoteIP:         remoteIP,
			RemoteMX:         req.RemoteMX,
			TLSVersion:       tlsVersion,
			TLSCipher:        tlsCipher,
			TLSPolicyApplied: tlsPolicyApplied,
			Duration:         time.Since(start),
		}, nil
	}

	// DATA
	wc, err := client.Data()
	if err != nil {
		code, enh := parseProtoError(err)
		return &DeliveryResponse{
			Success:          false,
			SMTPCode:         code,
			EnhancedCode:     enh,
			ResponseText:     err.Error(),
			RemoteIP:         remoteIP,
			RemoteMX:         req.RemoteMX,
			TLSVersion:       tlsVersion,
			TLSCipher:        tlsCipher,
			TLSPolicyApplied: tlsPolicyApplied,
			Duration:         time.Since(start),
		}, nil
	}

	if _, err := wc.Write(req.MIMEData); err != nil {
		_ = wc.Close()
		return nil, err
	}

	if err := wc.Close(); err != nil {
		code, enh := parseProtoError(err)
		return &DeliveryResponse{
			Success:          false,
			SMTPCode:         code,
			EnhancedCode:     enh,
			ResponseText:     err.Error(),
			RemoteIP:         remoteIP,
			RemoteMX:         req.RemoteMX,
			TLSVersion:       tlsVersion,
			TLSCipher:        tlsCipher,
			TLSPolicyApplied: tlsPolicyApplied,
			Duration:         time.Since(start),
		}, nil
	}

	_ = client.Quit()

	return &DeliveryResponse{
		Success:          true,
		SMTPCode:         "250",
		EnhancedCode:     "2.0.0",
		ResponseText:     "250 2.0.0 OK: message delivered",
		RemoteIP:         remoteIP,
		RemoteMX:         req.RemoteMX,
		TLSVersion:       tlsVersion,
		TLSCipher:        tlsCipher,
		TLSPolicyApplied: tlsPolicyApplied,
		Duration:         time.Since(start),
	}, nil
}

type RelayTransport struct {
	DialTimeout time.Duration
	IOTimeout   time.Duration
}

func NewRelayTransport() *RelayTransport {
	return &RelayTransport{
		DialTimeout: 30 * time.Second,
		IOTimeout:   60 * time.Second,
	}
}

type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		return []byte(a.password), nil
	}
	return nil, nil
}

func (r *RelayTransport) Deliver(ctx context.Context, req *DeliveryRequest) (*DeliveryResponse, error) {
	start := time.Now()
	if req.Relay == nil {
		return nil, errors.New("relay configuration missing for relay transport")
	}

	relay := req.Relay
	port := relay.Port
	if port <= 0 {
		port = 587
	}
	targetAddr := fmt.Sprintf("%s:%d", relay.Host, port)

	var dialer net.Dialer
	dialer.Timeout = r.DialTimeout
	conn, err := dialer.DialContext(ctx, "tcp", targetAddr)
	if err != nil {
		return &DeliveryResponse{
			Success:      false,
			SMTPCode:     "451",
			EnhancedCode: "4.4.1",
			ResponseText: fmt.Sprintf("relay connection failed to %s: %v", targetAddr, err),
			RemoteMX:     relay.Host,
			Duration:     time.Since(start),
		}, nil
	}
	defer conn.Close()

	remoteIP := ""
	if tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		remoteIP = tcpAddr.IP.String()
	}

	_ = conn.SetDeadline(time.Now().Add(r.IOTimeout))

	client, err := smtp.NewClient(conn, relay.Host)
	if err != nil {
		code, enh := parseProtoError(err)
		return &DeliveryResponse{
			Success:      false,
			SMTPCode:     code,
			EnhancedCode: enh,
			ResponseText: fmt.Sprintf("relay greeting error: %v", err),
			RemoteIP:     remoteIP,
			RemoteMX:     relay.Host,
			Duration:     time.Since(start),
		}, nil
	}
	defer client.Close()

	if err := client.Hello("mailvex.relay"); err != nil {
		code, enh := parseProtoError(err)
		return &DeliveryResponse{
			Success:      false,
			SMTPCode:     code,
			EnhancedCode: enh,
			ResponseText: fmt.Sprintf("relay EHLO rejected: %v", err),
			RemoteIP:     remoteIP,
			RemoteMX:     relay.Host,
			Duration:     time.Since(start),
		}, nil
	}

	// STARTTLS if configured
	if relay.TLSMode == "starttls" || relay.TLSMode == "" {
		if hasTLS, _ := client.Extension("STARTTLS"); hasTLS {
			_ = client.StartTLS(&tls.Config{ServerName: relay.Host})
		}
	}

	// Authentication
	if relay.AuthUser != nil && *relay.AuthUser != "" {
		authUser := *relay.AuthUser
		authPass := req.RelayAuthPass
		auth := smtp.PlainAuth("", authUser, authPass, relay.Host)
		if err := client.Auth(auth); err != nil {
			code, enh := parseProtoError(err)
			return &DeliveryResponse{
				Success:      false,
				SMTPCode:     code,
				EnhancedCode: enh,
				ResponseText: fmt.Sprintf("relay auth failed: %v", err),
				RemoteIP:     remoteIP,
				RemoteMX:     relay.Host,
				Duration:     time.Since(start),
			}, nil
		}
	}

	// From rewrite policy (§8.4)
	mimeToSend := req.MIMEData
	if relay.FromRewritePolicy == "replace" && relay.FromRewriteAddress != nil && *relay.FromRewriteAddress != "" {
		mimeToSend = rewriteFromHeader(mimeToSend, *relay.FromRewriteAddress)
	}

	mailFrom := req.ReturnPath
	if relay.FromRewritePolicy == "replace" && relay.FromRewriteAddress != nil && *relay.FromRewriteAddress != "" {
		mailFrom = *relay.FromRewriteAddress
	}

	if err := client.Mail(mailFrom); err != nil {
		code, enh := parseProtoError(err)
		return &DeliveryResponse{
			Success:      false,
			SMTPCode:     code,
			EnhancedCode: enh,
			ResponseText: err.Error(),
			RemoteIP:     remoteIP,
			RemoteMX:     relay.Host,
			Duration:     time.Since(start),
		}, nil
	}

	if err := client.Rcpt(req.RecipientAddr); err != nil {
		code, enh := parseProtoError(err)
		return &DeliveryResponse{
			Success:      false,
			SMTPCode:     code,
			EnhancedCode: enh,
			ResponseText: err.Error(),
			RemoteIP:     remoteIP,
			RemoteMX:     relay.Host,
			Duration:     time.Since(start),
		}, nil
	}

	wc, err := client.Data()
	if err != nil {
		code, enh := parseProtoError(err)
		return &DeliveryResponse{
			Success:      false,
			SMTPCode:     code,
			EnhancedCode: enh,
			ResponseText: err.Error(),
			RemoteIP:     remoteIP,
			RemoteMX:     relay.Host,
			Duration:     time.Since(start),
		}, nil
	}

	if _, err := wc.Write(mimeToSend); err != nil {
		_ = wc.Close()
		return nil, err
	}

	if err := wc.Close(); err != nil {
		code, enh := parseProtoError(err)
		return &DeliveryResponse{
			Success:      false,
			SMTPCode:     code,
			EnhancedCode: enh,
			ResponseText: err.Error(),
			RemoteIP:     remoteIP,
			RemoteMX:     relay.Host,
			Duration:     time.Since(start),
		}, nil
	}

	_ = client.Quit()

	return &DeliveryResponse{
		Success:      true,
		SMTPCode:     "250",
		EnhancedCode: "2.0.0",
		ResponseText: "250 2.0.0 OK: relay delivery complete",
		RemoteIP:     remoteIP,
		RemoteMX:     relay.Host,
		Duration:     time.Since(start),
	}, nil
}

func parseProtoError(err error) (string, string) {
	if err == nil {
		return "250", "2.0.0"
	}
	var tpErr *textproto.Error
	if errors.As(err, &tpErr) {
		code := fmt.Sprintf("%d", tpErr.Code)
		enh := extractEnhancedCode(tpErr.Msg)
		return code, enh
	}
	msg := err.Error()
	parts := strings.Fields(msg)
	code := "451"
	if len(parts) > 0 {
		if _, e := strconv.Atoi(parts[0]); e == nil {
			code = parts[0]
		}
	}
	return code, extractEnhancedCode(msg)
}

func extractEnhancedCode(response string) string {
	parts := strings.Fields(response)
	for _, p := range parts {
		if (strings.HasPrefix(p, "2.") || strings.HasPrefix(p, "4.") || strings.HasPrefix(p, "5.")) && strings.Count(p, ".") == 2 {
			return p
		}
	}
	return ""
}

func rewriteFromHeader(rawMIME []byte, newFrom string) []byte {
	reader := textproto.NewReader(bufio.NewReader(bytes.NewReader(rawMIME)))
	hdr, err := reader.ReadMIMEHeader()
	if err != nil {
		return rawMIME
	}

	var body bytes.Buffer
	for {
		line, err := reader.ReadLineBytes()
		if err != nil {
			break
		}
		body.Write(line)
		body.WriteString("\r\n")
	}

	origFrom := hdr.Get("From")
	hdr.Set("X-Original-From", origFrom)
	if hdr.Get("Reply-To") == "" {
		hdr.Set("Reply-To", origFrom)
	}
	hdr.Set("From", newFrom)

	var out bytes.Buffer
	for k, vv := range hdr {
		for _, v := range vv {
			out.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}
	}
	out.WriteString("\r\n")
	out.Write(body.Bytes())
	return out.Bytes()
}

func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLSv1.0"
	case tls.VersionTLS11:
		return "TLSv1.1"
	case tls.VersionTLS12:
		return "TLSv1.2"
	case tls.VersionTLS13:
		return "TLSv1.3"
	default:
		return fmt.Sprintf("TLS(0x%04x)", version)
	}
}
