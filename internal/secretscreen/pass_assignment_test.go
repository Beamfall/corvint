package secretscreen

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestStoredV1PatternIdentity(t *testing.T) {
	t.Run("LTA-V0-004 frozen-stored-pattern", func(t *testing.T) {
		const want = "6881463ef1568e369982842c14947be917ab7ad25f8a9a5f7e05581bd6511ff7"
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(StoredV1Pattern.String()))); got != want {
			t.Fatalf("stored-v1 pattern digest = %s, want %s", got, want)
		}
	})
}

func TestPassAssignmentBoundary(t *testing.T) {
	for _, test := range []struct {
		name, text, want string
	}{
		{name: "LTA-V0-004 public-validator-success", text: "Plugin validation passed: /private/tmp/corvint-log-admission-20260915/integrations/codex/plugins/corvint\n", want: "Plugin validation passed: /private/tmp/corvint-log-admission-20260915/integrations/codex/plugins/corvint\n"},
		{name: "LTA-V0-004 benign-bare-quoted-field", text: `passed = "all checks" status = "ok"`, want: `passed = "all checks" status = "ok"`},
		{name: "LTA-V0-004 benign-json-field", text: `{"passed":"all checks","status":"ok"}`, want: `{"passed":"all checks","status":"ok"}`},
		{name: "LTA-V0-004 benign-json-number", text: `{"passed":123,"status":"ok"}`, want: `{"passed":123,"status":"ok"}`},
		{name: "LTA-V0-004 pass-direct", text: `pass=synthetic123`, want: Placeholder},
		{name: "LTA-V0-004 prefixed-pass-direct", text: `DB_PASS: synthetic123`, want: Placeholder},
		{name: "LTA-V0-004 password-direct", text: `password=synthetic123`, want: Placeholder},
		{name: "LTA-V0-004 passphrase-direct", text: `passphrase=synthetic123`, want: Placeholder},
		{name: "LTA-V0-004 passwd-direct", text: `passwd=synthetic123`, want: Placeholder},
		{name: "LTA-V0-004 other-vocabulary-suffix", text: `db_password_hash=synthetic123`, want: Placeholder},
		{name: "LTA-V0-004 pass-double-quoted", text: `pass="synthetic 123 value" status="ok"`, want: Placeholder + ` status="ok"`},
		{name: "LTA-V0-004 prefixed-pass-single-quoted", text: `DB_PASS='synthetic 123 value' status="ok"`, want: Placeholder + ` status="ok"`},
		{name: "LTA-V0-004 password-single-quoted", text: `password='synthetic 123 value' status="ok"`, want: Placeholder + ` status="ok"`},
		{name: "LTA-V0-004 passphrase-double-quoted", text: `passphrase="synthetic 123 value" status="ok"`, want: Placeholder + ` status="ok"`},
		{name: "LTA-V0-004 passwd-double-quoted", text: `passwd="synthetic 123 value" status="ok"`, want: Placeholder + ` status="ok"`},
		{name: "LTA-V0-004 pass-json", text: `{"pass":"synthetic 123 value","status":"ok"}`, want: `{` + Placeholder + `,"status":"ok"}`},
		{name: "LTA-V0-004 prefixed-pass-json", text: `{"DB_PASS":"synthetic 123 value","status":"ok"}`, want: `{` + Placeholder + `,"status":"ok"}`},
		{name: "LTA-V0-004 password-json", text: `{"password":"synthetic 123 value","status":"ok"}`, want: `{` + Placeholder + `,"status":"ok"}`},
		{name: "LTA-V0-004 passphrase-json", text: `{"passphrase":"synthetic 123 value","status":"ok"}`, want: `{` + Placeholder + `,"status":"ok"}`},
		{name: "LTA-V0-004 passwd-json", text: `{"passwd":"synthetic 123 value","status":"ok"}`, want: `{` + Placeholder + `,"status":"ok"}`},
		{name: "LTA-V0-004 pass-escaped-value", text: `pass="synthetic \"123\" value" status="ok"`, want: Placeholder + ` status="ok"`},
		{name: "LTA-V0-004 pass-unterminated-value", text: "pass=\"synthetic 123\nvalue\\", want: Placeholder},
		{name: "LTA-V0-004 pass-json-unterminated-value", text: "{\"pass\":\"synthetic 123\nvalue\\", want: `{` + Placeholder},
		{name: "LTA-V0-004 pass-flag", text: `cmd --pass synthetic123`, want: `cmd ` + Placeholder},
		{name: "LTA-V0-004 pass-flag-quoted", text: `cmd --pass "synthetic 123 value" --verbose`, want: `cmd ` + Placeholder + ` --verbose`},
		{name: "LTA-V0-004 pass-flag-joined", text: `cmd --pass=synthetic123`, want: `cmd ` + Placeholder},
		{name: "LTA-V0-004 prefixed-pass-flag", text: `cmd --db-pass synthetic123`, want: `cmd ` + Placeholder},
	} {
		t.Run(test.name, func(t *testing.T) {
			screened, hit := Screen(test.text)
			wantHit := test.want != test.text
			if screened != test.want || hit != wantHit {
				t.Fatalf("Screen(%q) = (%q, %v), want (%q, %v)", test.text, screened, hit, test.want, wantHit)
			}
			if got := MatchString(test.text); got != wantHit {
				t.Fatalf("MatchString(%q) = %v, want %v", test.text, got, wantHit)
			}
		})
	}
}
