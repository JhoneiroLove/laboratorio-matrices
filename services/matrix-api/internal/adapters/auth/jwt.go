package auth

import (
	"crypto/rsa"
	"crypto/subtle"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/interseguro/matrix-api/internal/config"
)

type Service struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
	audience   string
	ttl        time.Duration
	leeway     time.Duration
	username   string
	password   string
	now        func() time.Time
}

func New(jwtConfig config.JWT, credentials config.DemoCredentials) (*Service, error) {
	privatePEM, err := os.ReadFile(jwtConfig.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer la clave privada JWT: %w", err)
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privatePEM)
	if err != nil {
		return nil, fmt.Errorf("no se pudo interpretar la clave privada JWT: %w", err)
	}
	publicPEM, err := os.ReadFile(jwtConfig.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer la clave pública JWT: %w", err)
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicPEM)
	if err != nil {
		return nil, fmt.Errorf("no se pudo interpretar la clave pública JWT: %w", err)
	}
	if privateKey.PublicKey.E != publicKey.E || privateKey.PublicKey.N.Cmp(publicKey.N) != 0 {
		return nil, fmt.Errorf("las claves privada y pública JWT no forman un par")
	}
	return &Service{
		privateKey: privateKey,
		publicKey:  publicKey,
		issuer:     jwtConfig.Issuer,
		audience:   jwtConfig.Audience,
		ttl:        jwtConfig.TTL,
		leeway:     jwtConfig.Leeway,
		username:   credentials.Username,
		password:   credentials.Password,
		now:        time.Now,
	}, nil
}

func (service *Service) Authenticate(username, password string) bool {
	usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(service.username))
	passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(service.password))
	return usernameMatch&passwordMatch == 1
}

func (service *Service) Issue(subject string) (string, int64, error) {
	now := service.now().UTC()
	expiresAt := now.Add(service.ttl)
	claims := jwt.RegisteredClaims{
		Issuer:    service.issuer,
		Subject:   subject,
		Audience:  jwt.ClaimStrings{service.audience},
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(service.privateKey)
	return token, int64(service.ttl / time.Second), err
}

func (service *Service) Verify(encoded string) error {
	token, err := jwt.ParseWithClaims(encoded, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, fmt.Errorf("método de firma inesperado")
		}
		return service.publicKey, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuer(service.issuer),
		jwt.WithAudience(service.audience),
		jwt.WithExpirationRequired(),
		jwt.WithNotBeforeRequired(),
		jwt.WithLeeway(service.leeway),
		jwt.WithTimeFunc(service.now),
	)
	if err != nil || !token.Valid {
		return fmt.Errorf("token inválido: %w", err)
	}
	return nil
}
