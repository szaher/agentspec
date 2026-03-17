package integration_tests

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/szaher/agentspec/internal/llm"
	"github.com/szaher/agentspec/internal/loop"
	"github.com/szaher/agentspec/internal/runtime"
	"github.com/szaher/agentspec/internal/session"
	"github.com/szaher/agentspec/internal/tools"
)

func TestTLSServerStartup(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	config := &runtime.RuntimeConfig{
		Agents: []runtime.AgentConfig{
			{Name: "tls-agent", Model: "test-model", Strategy: "react", MaxTurns: 5},
		},
	}

	mockClient := llm.NewMockClient(llm.MockResponse{
		Content:    "Hello!",
		StopReason: llm.StopEndTurn,
		Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 5},
	})

	registry := tools.NewRegistry()
	mgr := session.NewManager(session.NewMemoryStore(0, 0), nil)
	strategy := &loop.ReActStrategy{}

	srv := runtime.NewServer(config, mockClient, registry, mgr, strategy,
		runtime.WithTLS(certFile, keyFile),
		runtime.WithNoAuth(true))

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe(addr)
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Connect with TLS client that trusts the test cert
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // test only
	}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		t.Fatalf("TLS dial failed: %v", err)
	}
	_ = conn.Close()
}

func TestTLSRejectsMissingCert(t *testing.T) {
	config := &runtime.RuntimeConfig{
		Agents: []runtime.AgentConfig{
			{Name: "test-agent", Model: "test-model"},
		},
	}

	registry := tools.NewRegistry()
	mgr := session.NewManager(session.NewMemoryStore(0, 0), nil)
	strategy := &loop.ReActStrategy{}

	srv := runtime.NewServer(config, nil, registry, mgr, strategy,
		runtime.WithTLS("/nonexistent/cert.pem", "/nonexistent/key.pem"),
		runtime.WithNoAuth(true))

	err := srv.ListenAndServe("127.0.0.1:0")
	if err == nil {
		t.Error("expected error for missing TLS cert, got nil")
	}
}

func TestTLSCertificateReload(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	config := &runtime.RuntimeConfig{
		Agents: []runtime.AgentConfig{
			{Name: "reload-agent", Model: "test-model"},
		},
	}

	registry := tools.NewRegistry()
	mgr := session.NewManager(session.NewMemoryStore(0, 0), nil)
	strategy := &loop.ReActStrategy{}

	srv := runtime.NewServer(config, nil, registry, mgr, strategy,
		runtime.WithTLS(certFile, keyFile),
		runtime.WithNoAuth(true))

	// Initial cert load should succeed
	err := srv.ReloadTLSCertificate()
	if err != nil {
		t.Errorf("initial reload should succeed: %v", err)
	}
}

// generateTestCert creates a self-signed cert and key in temp dir, returns paths.
func generateTestCert(t *testing.T) (certPath, keyPath string) {
	t.Helper()
	dir := t.TempDir()
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Test"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		t.Fatalf("write cert: %v", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyPath, keyPEM, 0644); err != nil {
		t.Fatalf("write key: %v", err)
	}

	return certPath, keyPath
}
