package email

import (
	"bufio"
	"fmt"
	"github.com/NookMux/NookMux/internal/common"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type smtpTestServer struct {
	listener net.Listener
	addr     string
	authCh   chan string
	once     sync.Once
}

func newSMTPTestServer(t *testing.T) *smtpTestServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error: %v", err)
	}
	server := &smtpTestServer{
		listener: listener,
		addr:     listener.Addr().String(),
		authCh:   make(chan string, 1),
	}
	go server.serve(t)
	t.Cleanup(server.close)
	return server
}

func (s *smtpTestServer) close() {
	s.once.Do(func() {
		_ = s.listener.Close()
	})
}

func (s *smtpTestServer) serve(t *testing.T) {
	t.Helper()
	conn, err := s.listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	writeLine := func(format string, args ...any) {
		_, _ = fmt.Fprintf(rw, format+"\r\n", args...)
		_ = rw.Flush()
	}
	readLine := func() (string, error) {
		line, err := rw.ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}

	writeLine("220 localhost ESMTP ready")
	for {
		line, err := readLine()
		if err != nil {
			return
		}
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO") || strings.HasPrefix(upper, "HELO"):
			writeLine("250-localhost")
			writeLine("250-AUTH PLAIN LOGIN")
			writeLine("250 OK")
		case strings.HasPrefix(upper, "AUTH LOGIN"):
			select {
			case s.authCh <- "AUTH LOGIN":
			default:
			}
			writeLine("334 VXNlcm5hbWU6")
			if _, err := readLine(); err != nil {
				return
			}
			writeLine("334 UGFzc3dvcmQ6")
			if _, err := readLine(); err != nil {
				return
			}
			writeLine("235 2.7.0 Authentication successful")
		case strings.HasPrefix(upper, "AUTH PLAIN"):
			select {
			case s.authCh <- "AUTH PLAIN":
			default:
			}
			writeLine("235 2.7.0 Authentication successful")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			writeLine("250 2.1.0 OK")
		case strings.HasPrefix(upper, "RCPT TO:"):
			writeLine("250 2.1.5 OK")
		case upper == "DATA":
			writeLine("354 End data with <CR><LF>.<CR><LF>")
			for {
				dataLine, err := readLine()
				if err != nil {
					return
				}
				if dataLine == "." {
					break
				}
			}
			writeLine("250 2.0.0 queued")
		case upper == "QUIT":
			writeLine("221 2.0.0 bye")
			return
		default:
			writeLine("250 OK")
		}
	}
}

func TestSendEmailUsesPlainAuthByDefault(t *testing.T) {
	server := newSMTPTestServer(t)
	prev := snapshotSMTPConfig()
	defer restoreSMTPConfig(prev)

	common.SMTPServer = "127.0.0.1"
	common.SMTPPort = portFromAddr(server.addr)
	common.SMTPFrom = "sender@example.com"
	common.SMTPAccount = "sender@example.com"
	common.SMTPToken = "secret"
	common.SMTPSSLEnabled = false
	common.SMTPForceLoginAuthEnabled = false

	if err := SendEmail("Test Subject", "receiver@example.com", "hello world"); err != nil {
		t.Fatalf("SendEmail() error: %v", err)
	}

	select {
	case auth := <-server.authCh:
		if auth != "AUTH PLAIN" {
			t.Fatalf("expected AUTH PLAIN, got %q", auth)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SMTP auth command")
	}
}

func TestSendEmailUsesLoginAuthWhenForced(t *testing.T) {
	server := newSMTPTestServer(t)
	prev := snapshotSMTPConfig()
	defer restoreSMTPConfig(prev)

	common.SMTPServer = "127.0.0.1"
	common.SMTPPort = portFromAddr(server.addr)
	common.SMTPFrom = "sender@example.com"
	common.SMTPAccount = "sender@example.com"
	common.SMTPToken = "secret"
	common.SMTPSSLEnabled = false
	common.SMTPForceLoginAuthEnabled = true

	if err := SendEmail("Test Subject", "receiver@example.com", "hello world"); err != nil {
		t.Fatalf("SendEmail() error: %v", err)
	}

	select {
	case auth := <-server.authCh:
		if auth != "AUTH LOGIN" {
			t.Fatalf("expected AUTH LOGIN, got %q", auth)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SMTP auth command")
	}
}

type smtpSnapshot struct {
	server         string
	port           int
	ssl            bool
	account        string
	from           string
	token          string
	forceLoginAuth bool
}

func snapshotSMTPConfig() smtpSnapshot {
	return smtpSnapshot{
		server:         common.SMTPServer,
		port:           common.SMTPPort,
		ssl:            common.SMTPSSLEnabled,
		account:        common.SMTPAccount,
		from:           common.SMTPFrom,
		token:          common.SMTPToken,
		forceLoginAuth: common.SMTPForceLoginAuthEnabled,
	}
}

func restoreSMTPConfig(snapshot smtpSnapshot) {
	common.SMTPServer = snapshot.server
	common.SMTPPort = snapshot.port
	common.SMTPSSLEnabled = snapshot.ssl
	common.SMTPAccount = snapshot.account
	common.SMTPFrom = snapshot.from
	common.SMTPToken = snapshot.token
	common.SMTPForceLoginAuthEnabled = snapshot.forceLoginAuth
}

func portFromAddr(addr string) int {
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0
	}
	port, _ := strconv.Atoi(portStr)
	return port
}
