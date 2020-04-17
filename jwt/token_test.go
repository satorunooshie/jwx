package jwt_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/jwt"
	"github.com/stretchr/testify/assert"
)

const (
	tokenTime = 233431200
)

var expectedTokenTime = time.Unix(tokenTime, 0).UTC()

func TestHeader(t *testing.T) {
	values := map[string]interface{}{
		jwt.AudienceKey:   []string{"developers", "secops", "tac"},
		jwt.ExpirationKey: expectedTokenTime,
		jwt.IssuedAtKey:   expectedTokenTime,
		jwt.IssuerKey:     "http://www.example.com",
		jwt.JwtIDKey:      "e9bc097a-ce51-4036-9562-d2ade882db0d",
		jwt.NotBeforeKey:  expectedTokenTime,
		jwt.SubjectKey:    "unit test",
	}

	t.Run("Roundtrip", func(t *testing.T) {
		var h jwt.Token
		for k, v := range values {
			err := h.Set(k, v)
			if err != nil {
				t.Fatalf("Set failed for %s", k)
			}
			got, ok := h.Get(k)
			if !ok {
				t.Fatalf("Set failed for %s", k)
			}
			if !reflect.DeepEqual(v, got) {
				t.Fatalf("Values do not match: (%v, %v)", v, got)
			}
		}
	})

	t.Run("RoundtripError", func(t *testing.T) {
		type dummyStruct struct {
			dummy1 int
			dummy2 float64
		}
		dummy := &dummyStruct{1, 3.4}

		values := map[string]interface{}{
			jwt.AudienceKey:   dummy,
			jwt.ExpirationKey: dummy,
			jwt.IssuedAtKey:   dummy,
			jwt.IssuerKey:     dummy,
			jwt.JwtIDKey:      dummy,
			jwt.NotBeforeKey:  dummy,
			jwt.SubjectKey:    dummy,
		}

		var h jwt.Token
		for k, v := range values {
			err := h.Set(k, v)
			if err == nil {
				t.Fatalf("Setting %s value should have failed", k)
			}
		}
		err := h.Set("default", dummy) // private params
		if err != nil {
			t.Fatalf("Setting %s value failed", "default")
		}
		for k := range values {
			_, ok := h.Get(k)
			if ok {
				t.Fatalf("Getting %s value should have failed", k)
			}
		}
		_, ok := h.Get("default")
		if !ok {
			t.Fatal("Failed to get default value")
		}
	})

	t.Run("GetError", func(t *testing.T) {
		var h jwt.Token
		issuer := h.Issuer()
		if issuer != "" {
			t.Fatalf("Get Issuer should return empty string")
		}
		jwtID := h.JwtID()
		if jwtID != "" {
			t.Fatalf("Get JWT Id should return empty string")
		}
	})
}

func TestTokenMarshal(t *testing.T) {
	t1 := jwt.New()
	err := t1.Set(jwt.JwtIDKey, "AbCdEfG")
	if err != nil {
		t.Fatalf("Failed to set JWT ID: %s", err.Error())
	}
	err = t1.Set(jwt.SubjectKey, "foobar@example.com")
	if err != nil {
		t.Fatalf("Failed to set Subject: %s", err.Error())
	}

	// Silly fix to remove monotonic element from time.Time obtained
	// from time.Now(). Without this, the equality comparison goes
	// ga-ga for golang tip (1.9)
	now := time.Unix(time.Now().Unix(), 0)
	err = t1.Set(jwt.IssuedAtKey, now.Unix())
	if err != nil {
		t.Fatalf("Failed to set IssuedAt: %s", err.Error())
	}
	err = t1.Set(jwt.NotBeforeKey, now.Add(5*time.Second))
	if err != nil {
		t.Fatalf("Failed to set NotBefore: %s", err.Error())
	}
	err = t1.Set(jwt.ExpirationKey, now.Add(10*time.Second).Unix())
	if err != nil {
		t.Fatalf("Failed to set Expiration: %s", err.Error())
	}
	err = t1.Set(jwt.AudienceKey, []string{"devops", "secops", "tac"})
	if err != nil {
		t.Fatalf("Failed to set audience: %s", err.Error())
	}
	err = t1.Set("custom", "MyValue")
	if err != nil {
		t.Fatalf(`Failed to set private claim "custom": %s`, err.Error())
	}
	jsonbuf1, err := json.MarshalIndent(t1, "", "  ")
	if err != nil {
		t.Fatalf("JSON Marshal failed: %s", err.Error())
	}

	t2 := jwt.New()
	err = json.Unmarshal(jsonbuf1, t2)
	if err != nil {
		t.Fatalf("JSON Unmarshal error: %s", err.Error())
	}

	if !assert.Equal(t, t1, t2, "tokens should match") {
		return
	}

	_, err = json.MarshalIndent(t2, "", "  ")
	if err != nil {
		t.Fatalf("JSON marshal error: %s", err.Error())
	}
}

func TestToken(t *testing.T) {
	tok := jwt.New()

	claims := map[string]interface{}{
		jwt.AudienceKey:   []string{"developers", "secops", "tac"},
		jwt.ExpirationKey: expectedTokenTime,
		jwt.IssuedAtKey:   expectedTokenTime,
		jwt.IssuerKey:     "http://www.example.com",
		jwt.JwtIDKey:      "e9bc097a-ce51-4036-9562-d2ade882db0d",
		jwt.NotBeforeKey:  expectedTokenTime,
		jwt.SubjectKey:    "unit test",
		"myClaim":         "hello, world",
	}
	for key, value := range claims {
		tok.Set(key, value)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	for iter := tok.Iterate(ctx); iter.Next(ctx); {
		pair := iter.Pair()
		t.Logf("%s -> %v", pair.Key, pair.Value)
	}

	m, err := tok.AsMap(ctx)
	if !assert.NoError(t, err, `AsMap should succeed`) {
		return
	}

	if !assert.Equal(t, m, claims, "hash should match") {
		return
	}
}
