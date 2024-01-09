package eztls

/*
 * cache_test.go
 * Tests for cache.go
 * By J. Stuart McMurray
 * Created 20231209
 * Last Modified 20231209
 */

import (
	"context"
	"reflect"
	"testing"

	"golang.org/x/crypto/acme/autocert"
)

/*
The first two test functions in this file were mooched from the
golang.org/x/crypto/acme/autocert sources, which is under the following
license:

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

func TestMemCache_CacheInterface(t *testing.T) {
	/*
		Mooched from:
		https://cs.opensource.google/go/x/crypto/+/refs/tags/v0.16.0:acme/autocert/cache_test.go
	*/

	// make sure DirCache satisfies Cache interface
	var _ autocert.Cache = autocert.DirCache("/")
}

func TestMemCache(t *testing.T) {
	cache := NewMemCache()

	/*
		Mooched from:
		https://cs.opensource.google/go/x/crypto/+/refs/tags/v0.16.0:acme/autocert/cache_test.go
	*/
	ctx := context.Background()

	// test cache miss
	if _, err := cache.Get(ctx, "nonexistent"); err != autocert.ErrCacheMiss {
		t.Errorf("get: %v; want ErrCacheMiss", err)
	}

	// test put/get
	b1 := []byte{1}
	if err := cache.Put(ctx, "dummy", b1); err != nil {
		t.Fatalf("put: %v", err)
	}
	b2, err := cache.Get(ctx, "dummy")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !reflect.DeepEqual(b1, b2) {
		t.Errorf("b1 = %v; want %v", b1, b2)
	}

	// test delete
	if err := cache.Delete(ctx, "dummy"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := cache.Get(ctx, "dummy"); err != autocert.ErrCacheMiss {
		t.Errorf("get: %v; want ErrCacheMiss", err)
	}
}

func TestDirCache_Smoketest(t *testing.T) {
	d := t.TempDir()
	if _, err := dirCache(d); nil != err {
		t.Fatalf("Error: %s", err)
	}
}
