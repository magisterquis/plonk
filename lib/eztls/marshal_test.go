package eztls

/*
 * marshal_test.go
 * Marshal and unmarshal certs for caching.
 * By J. Stuart McMurray
 * Created 20231209
 * Last Modified 20231209
 */

import "testing"

func TestMarshalCertificateUnmarshalCertificate(t *testing.T) {
	have, err := GenerateSelfSignedCertificate(
		"kittens",
		nil,
		nil,
		0,
	)
	if nil != err {
		t.Fatalf("Error generating cert: %s", err)
	}

	b, err := MarshalCertificate(have)
	if nil != err {
		t.Fatalf("MarshalCertificate failed: %s", err)
	}
	if 0 == len(b) {
		t.Fatalf("Marshalled certificate slice empty")
	}

	got, err := UnmarshalCertificate(b)
	if nil != err {
		t.Fatalf("UnmarshalCertificate failed: %s", err)
	}

	if !got.Leaf.Equal(have.Leaf) {
		t.Fatalf("Unmarshalled certificate incorrect")
	}
}
