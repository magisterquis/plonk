package eztls

/*
 * marshal.go
 * Marshal and unmarshal certs for caching.
 * By J. Stuart McMurray
 * Created 20231209
 * Last Modified 20231209
 */

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"
)

/*
Most of this code mooched from golang.org/x/crypto/acme/autocert sources,
which is under the following license:

Copyright (c) 2009 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// MarshalCertificate marshals the certificate c; it is the inverse of
// UnmarshalCertificate.
func MarshalCertificate(c *tls.Certificate) ([]byte, error) {
	/*
		Mooched from
		https://cs.opensource.google/go/x/crypto/+/refs/tags/v0.16.0:acme/autocert/autocert.go
	*/

	// contains PEM-encoded data
	var buf bytes.Buffer

	// private
	switch key := c.PrivateKey.(type) {
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf(
				"encoding ECDSA private key: %w",
				err,
			)
		}
		pb := &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
		if err := pem.Encode(&buf, pb); nil != err {
			return nil, fmt.Errorf(
				"encoding ECDSA private key: %w",
				err,
			)
		}
	case *rsa.PrivateKey:
		b := x509.MarshalPKCS1PrivateKey(key)
		pb := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: b}
		if err := pem.Encode(&buf, pb); err != nil {
			return nil, fmt.Errorf(
				"encoding RSA private key: %w",
				err,
			)
		}
	default:
		return nil, errors.New("unknown private key type")
	}

	// public
	for _, b := range c.Certificate {
		pb := &pem.Block{Type: "CERTIFICATE", Bytes: b}
		if err := pem.Encode(&buf, pb); err != nil {
			return nil, fmt.Errorf(
				"encoding certificate: %w",
				err,
			)
		}
	}

	return buf.Bytes(), nil
}

// UnmarshalCertificate unmarshals the certificate in b; it is the inverse of
// MarshalCertificate.
func UnmarshalCertificate(b []byte) (*tls.Certificate, error) {
	return unmarshalCert(b, time.Now())
}

// unmarshalCert is like UnmarshalCertificate, but allows setting the time for
// testing.
func unmarshalCert(b []byte, now time.Time) (*tls.Certificate, error) {
	/*
		Mooched from
		https://cs.opensource.google/go/x/crypto/+/refs/tags/v0.16.0:acme/autocert/autocert.go
	*/

	// private
	priv, pub := pem.Decode(b)
	if priv == nil || !strings.Contains(priv.Type, "PRIVATE") {
		return nil, fmt.Errorf("PRIVATE not found")
	}
	privKey, err := parsePrivateKey(priv.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	// public
	var pubDER [][]byte
	for len(pub) > 0 {
		var b *pem.Block
		b, pub = pem.Decode(pub)
		if b == nil {
			break
		}
		pubDER = append(pubDER, b.Bytes)
	}
	if len(pub) > 0 {
		// Leftover content not consumed by pem.Decode. Corrupt. Ignore.
		return nil, errors.New("excess data after certificate")
	}

	// verify and create TLS cert
	leaf, err := validCert(pubDER, privKey, now)
	if err != nil {
		return nil, fmt.Errorf("invalid cert: %w", err)
	}
	tlscert := &tls.Certificate{
		Certificate: pubDER,
		PrivateKey:  privKey,
		Leaf:        leaf,
	}
	return tlscert, nil

}

// validCert parses a cert chain provided as der argument and verifies the leaf and der[0]
// correspond to the private key, the domain and key type match, and expiration dates
// are valid. It doesn't do any revocation checking.
//
// The returned value is the verified leaf cert.
func validCert(der [][]byte, key crypto.Signer, now time.Time) (leaf *x509.Certificate, err error) {
	/*
		Mooched from
		https://cs.opensource.google/go/x/crypto/+/refs/tags/v0.16.0:acme/autocert/autocert.go
	*/

	// parse public part(s)
	var n int
	for _, b := range der {
		n += len(b)
	}
	pub := make([]byte, n)
	n = 0
	for _, b := range der {
		n += copy(pub[n:], b)
	}
	x509Cert, err := x509.ParseCertificates(pub)
	if nil != err {
		return nil, fmt.Errorf("parsing x509 certificate: %w", err)
	} else if len(x509Cert) == 0 {
		return nil, errors.New("no public key found")
	}
	// verify the leaf is not expired and matches the domain name
	leaf = x509Cert[0]
	if now.Before(leaf.NotBefore) {
		return nil, errors.New("certificate not valid yet")
	}
	if now.After(leaf.NotAfter) {
		return nil, errors.New("certificate expired")
	}
	// ensure the leaf corresponds to the private key and matches the certKey type
	switch pub := leaf.PublicKey.(type) {
	case *rsa.PublicKey:
		prv, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("private/public key type mismatch")
		}
		if pub.N.Cmp(prv.N) != 0 {
			return nil, errors.New("private/public key mismatch")
		}
	case *ecdsa.PublicKey:
		prv, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("private/public key type mismatch")
		}
		if pub.X.Cmp(prv.X) != 0 || pub.Y.Cmp(prv.Y) != 0 {
			return nil, errors.New("private/public key mismatch")
		}
	default:
		return nil, errors.New("unknown public key algorithm")
	}
	return leaf, nil
}

// Attempt to parse the given private key DER block. OpenSSL 0.9.8 generates
// PKCS#1 private keys by default, while OpenSSL 1.0.0 generates PKCS#8 keys.
// OpenSSL ecparam generates SEC1 EC private keys for ECDSA. We try all three.
//
// Inspired by parsePrivateKey in crypto/tls/tls.go.
func parsePrivateKey(der []byte) (crypto.Signer, error) {
	/*
		Mooched from
		https://cs.opensource.google/go/x/crypto/+/refs/tags/v0.16.0:acme/autocert/autocert.go
	*/

	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey:
			return key, nil
		case *ecdsa.PrivateKey:
			return key, nil
		default:
			return nil, errors.New(
				"unknown PCKS#8 private key type",
			)
		}
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}

	return nil, errors.New("failed to guess form")
}
