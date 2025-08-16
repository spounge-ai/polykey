package domain

import (
	"context"
	"crypto/x509"
)

type peerCertKey struct{}

// NewContextWithPeerCert creates a new context with the peer's leaf certificate.
func NewContextWithPeerCert(ctx context.Context, cert *x509.Certificate) context.Context {
	return context.WithValue(ctx, peerCertKey{}, cert)
}

// PeerCertFromContext retrieves the client's leaf certificate from the context.
func PeerCertFromContext(ctx context.Context) (*x509.Certificate, bool) {
	cert, ok := ctx.Value(peerCertKey{}).(*x509.Certificate)
	return cert, ok
}
