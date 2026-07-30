package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/interseguro/matrix-api/internal/config"
)

func TestNewRejectsMismatchedKeyPair(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("no se pudo generar la clave privada: %v", err)
	}
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("no se pudo generar la segunda clave: %v", err)
	}
	directory := t.TempDir()
	privatePath := filepath.Join(directory, "private.pem")
	publicPath := filepath.Join(directory, "public.pem")
	privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("no se pudo serializar la clave privada: %v", err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&otherKey.PublicKey)
	if err != nil {
		t.Fatalf("no se pudo serializar la clave pública: %v", err)
	}
	if err := os.WriteFile(privatePath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER}), 0o600); err != nil {
		t.Fatalf("no se pudo escribir la clave privada: %v", err)
	}
	if err := os.WriteFile(publicPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER}), 0o600); err != nil {
		t.Fatalf("no se pudo escribir la clave pública: %v", err)
	}

	_, err = New(config.JWT{
		PrivateKeyPath: privatePath, PublicKeyPath: publicPath,
		Issuer: "matrix-api", Audience: "matrix-api", TTL: time.Minute,
	}, config.DemoCredentials{Username: "demo", Password: "secret"})
	if err == nil {
		t.Fatal("New() devolvió nil; se esperaba el rechazo de claves que no forman un par")
	}
}

func TestVerifyRegisteredClaimsAndAlgorithm(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("no se pudo generar la clave RSA: %v", err)
	}
	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	service := &Service{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		issuer:     "matrix-api",
		audience:   "matrix-clients",
		ttl:        time.Minute,
		now:        func() time.Time { return now },
	}

	validClaims := func() jwt.RegisteredClaims {
		return jwt.RegisteredClaims{
			Issuer:    "matrix-api",
			Audience:  jwt.ClaimStrings{"matrix-clients"},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			NotBefore: jwt.NewNumericDate(now.Add(-time.Second)),
		}
	}
	tests := []struct {
		name   string
		claims jwt.RegisteredClaims
		method jwt.SigningMethod
		key    any
		valid  bool
	}{
		{name: "válido", claims: validClaims(), method: jwt.SigningMethodRS256, key: privateKey, valid: true},
		{name: "issuer incorrecto", claims: withIssuer(validClaims(), "other"), method: jwt.SigningMethodRS256, key: privateKey},
		{name: "audience incorrecto", claims: withAudience(validClaims(), "other"), method: jwt.SigningMethodRS256, key: privateKey},
		{name: "expirado", claims: withExpiration(validClaims(), now.Add(-time.Second)), method: jwt.SigningMethodRS256, key: privateKey},
		{name: "aún no activo", claims: withNotBefore(validClaims(), now.Add(time.Second)), method: jwt.SigningMethodRS256, key: privateKey},
		{name: "algoritmo incorrecto", claims: validClaims(), method: jwt.SigningMethodHS256, key: []byte("not-an-rsa-key")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := jwt.NewWithClaims(test.method, test.claims).SignedString(test.key)
			if err != nil {
				t.Fatalf("no se pudo firmar el token: %v", err)
			}
			err = service.Verify(encoded)
			if test.valid && err != nil {
				t.Fatalf("Verify() devolvió un error inesperado: %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("Verify() devolvió nil; se esperaba el rechazo")
			}
		})
	}
}

func withIssuer(claims jwt.RegisteredClaims, issuer string) jwt.RegisteredClaims {
	claims.Issuer = issuer
	return claims
}

func withAudience(claims jwt.RegisteredClaims, audience string) jwt.RegisteredClaims {
	claims.Audience = jwt.ClaimStrings{audience}
	return claims
}

func withExpiration(claims jwt.RegisteredClaims, expiration time.Time) jwt.RegisteredClaims {
	claims.ExpiresAt = jwt.NewNumericDate(expiration)
	return claims
}

func withNotBefore(claims jwt.RegisteredClaims, notBefore time.Time) jwt.RegisteredClaims {
	claims.NotBefore = jwt.NewNumericDate(notBefore)
	return claims
}
