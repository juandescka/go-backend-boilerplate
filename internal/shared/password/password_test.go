package password

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestArgon2idPolicyAndVerification(t *testing.T) {
	secret := uuid.NewString()
	hash, err := Hash(secret)
	if err != nil || !strings.HasPrefix(hash, phcPrefix) {
		t.Fatal("hash generation failed")
	}
	if !Verify(hash, secret) || Verify(hash, "incorrect") {
		t.Fatal("password comparison incorrect")
	}
	other, err := Hash(secret)
	if err != nil || other == hash {
		t.Fatal("independent salts required")
	}
	for _, invalid := range []string{"", "not-a-hash", strings.Replace(hash, "m=65536", "m=4294967295", 1), strings.Replace(hash, "v=19", "v=16", 1), hash + "$extra"} {
		if Verify(invalid, secret) {
			t.Fatal("invalid/missing/unsupported hash accepted")
		}
	}
	for _, invalid := range []string{"", strings.Repeat("a", MaxBytes+1), string([]byte{0xff})} {
		if _, err := Hash(invalid); err == nil || Verify(hash, invalid) {
			t.Fatal("malformed password accepted")
		}
	}
}

func TestPHCDecoderBounds(t *testing.T) {
	for _, encoded := range []string{phcPrefix, phcPrefix + strings.Repeat("A", 500), "$argon2id$v=19$m=0,t=0,p=0$AA$AA", phcPrefix + "AA$AA"} {
		if _, _, ok := decode(encoded); ok {
			t.Fatal("unbounded or truncated PHC accepted")
		}
	}
}

func BenchmarkVerify(b *testing.B) {
	hash, err := Hash("benchmark input only")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !Verify(hash, "benchmark input only") {
			b.Fatal("verification failed")
		}
	}
}
