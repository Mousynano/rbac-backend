// Package mail sends transactional messages through a configured SMTP relay.
package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Config describes one SMTP endpoint.
type Config struct {
	Host                              string
	Port                              int
	Username, Password, From, TLSMode string
}

// SMTP sends plain text mail through STARTTLS or implicit TLS.
type SMTP struct{ Config Config }

// Send delivers a message with a bounded network deadline.
func (s SMTP) Send(ctx context.Context, to, subject, body string) error {
	address := net.JoinHostPort(s.Config.Host, fmt.Sprint(s.Config.Port))
	dialer := net.Dialer{Timeout: 10 * time.Second}
	var conn net.Conn
	var err error
	if s.Config.TLSMode == "implicit" {
		raw, dialErr := dialer.DialContext(ctx, "tcp", address)
		if dialErr != nil {
			return fmt.Errorf("connect smtp: %w", dialErr)
		}
		tlsConn := tls.Client(raw, &tls.Config{ServerName: s.Config.Host, MinVersion: tls.VersionTLS12})
		if deadline, ok := ctx.Deadline(); ok {
			_ = tlsConn.SetDeadline(deadline)
		}
		if err = tlsConn.HandshakeContext(ctx); err != nil {
			_ = raw.Close()
			return fmt.Errorf("smtp tls handshake: %w", err)
		}
		conn = tlsConn
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return fmt.Errorf("connect smtp: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	}
	client, err := smtp.NewClient(conn, s.Config.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()
	if s.Config.TLSMode == "starttls" {
		if err = client.StartTLS(&tls.Config{ServerName: s.Config.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if s.Config.Username != "" {
		if err = client.Auth(smtp.PlainAuth("", s.Config.Username, s.Config.Password, s.Config.Host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err = client.Mail(s.Config.From); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	subject = strings.NewReplacer("\r", "", "\n", " ").Replace(subject)
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s", s.Config.From, to, subject, body)
	if _, err = w.Write([]byte(message)); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return client.Quit()
}
