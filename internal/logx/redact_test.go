package logx

import (
	"strings"
	"testing"
)

func TestRedactMasksSecrets(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		secrets []string
	}{
		{"https pair link", "open https://device.example.invalid/pair#token=faketoken0000placeholder now", []string{"faketoken0000placeholder", "/pair#"}},
		{"schemeless pair link", "go to /pair#token=faketoken0000placeholder", []string{"faketoken0000placeholder", "/pair#"}},
		{"query token", "GET /x?token=faketoken0000placeholder", []string{"faketoken0000placeholder"}},
		{"json token", `{"token":"FAKE-TOKEN-0000-placeholder"}`, []string{"FAKE-TOKEN-0000-placeholder"}},
		{"json code string", `{"code":"FAKE-1234"}`, []string{"FAKE-1234"}},
		{"json code number", `{"code":123456}`, []string{"123456"}},
		{"json pairing_code", `{"pairing_code":"FAKEPAIR"}`, []string{"FAKEPAIR"}},
		{"json pairingCode", `{"pairingCode":"FAKEPAIR"}`, []string{"FAKEPAIR"}},
		{"json pair_code", `{"pair_code":"FAKEPAIR"}`, []string{"FAKEPAIR"}},
		{"escaped json", `body="{\"token\":\"FAKE-TOKEN-0000-placeholder\"}"`, []string{"FAKE-TOKEN-0000-placeholder"}},
		{"logfmt token", "msg=hi token=FAKE-TOKEN-0000-placeholder ok=1", []string{"FAKE-TOKEN-0000-placeholder"}},
		{"logfmt quoted pairing_code", `pairing_code="FAKE PAIR 1"`, []string{"FAKE PAIR 1"}},
		{"bare grouped code", "Your code is ABCD-1234 for now", []string{"ABCD-1234"}},
		{"bare three-group code", "Enter AB12-CD34-EF56 on the device", []string{"AB12-CD34-EF56"}},
		{"long token-like string", "value FAKEtoken0000placeholderFAKEtoken0000 end", []string{"FAKEtoken0000placeholderFAKEtoken0000"}},
		{"json-escaped ampersand link", `{"pair_url":"https://device.example.invalid/pair#v=1\u0026token=FAKESECRET"}`, []string{"FAKESECRET"}},
		{"json-escaped ampersand query", `{"url":"https://device.example.invalid/x?a=1\u0026token=FAKESECRET"}`, []string{"FAKESECRET"}},
		{"percent-encoded pair link", "next=https%3A%2F%2Fdevice.example.invalid%2Fpair%23token%3DFAKESECRET", []string{"FAKESECRET"}},
		{"go struct output", "{Token:FAKESECRET F:<nil>}", []string{"FAKESECRET"}},
		{"double-escaped nested json", `b="{\"data\":\"{\\\"token\\\":\\\"FAKENESTED\\\"}\"}"`, []string{"FAKENESTED"}},
		{"colon form", "pairing code: FAKESECRET", []string{"FAKESECRET"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Redact(tc.in)
			if !strings.Contains(got, Redacted) {
				t.Errorf("Redact(%q) = %q, missing %q", tc.in, got, Redacted)
			}
			for _, s := range tc.secrets {
				if strings.Contains(got, s) {
					t.Errorf("Redact(%q) = %q, still contains %q", tc.in, got, s)
				}
			}
		})
	}
}

func TestRedactKeepsHarmlessText(t *testing.T) {
	for _, in := range []string{
		"Connecting to device laptop over ssh",
		"exit_code=1 status=200",
		"exit code:\nnext-line-word follows",
		"device 550e8400-e29b-41d4-a716-446655440000 seen",
		"commit 0123456789abcdef0123456789abcdef01234567",
		"version v0.1.0-dev built 2026-11-24",
		"UTF-8 and SHA-256 are fine",
		"the pairing flow started",
		"https://example.invalid/devices/list",
		"error code path",
		"my-device-name-here",
		"internationalization",
		"a URL like https://example.invalid/docs#install",
	} {
		t.Run(in, func(t *testing.T) {
			if got := Redact(in); got != in {
				t.Errorf("Redact(%q) = %q, want unchanged", in, got)
			}
		})
	}
}

func TestRedactIdempotent(t *testing.T) {
	for _, in := range []string{
		`{"token":"FAKE-TOKEN-0000-placeholder","code":"FAKE-1234","expires_in":300}`,
		"open https://device.example.invalid/pair#token=faketoken0000placeholder code ABCD-1234",
		"the pairing flow started",
		"msg=hi token=[REDACTED] pairing_code=[REDACTED]",
		"{Token:[REDACTED] code: [REDACTED]}",
	} {
		once := Redact(in)
		if twice := Redact(once); twice != once {
			t.Errorf("Redact not idempotent for %q: once=%q twice=%q", in, once, twice)
		}
	}
}

func TestIsBareCode(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"ABCD-1234", true},
		{"AB12-CD34-EF56", true},
		{"ABCD1234EFGH5678ABCD1234EFGH5678", true},
		{"ABCDEFGH-ABCDEFGH", false},
		{"1234-5678", false},
		{"abcd-1234", false},
		{"ABC-123", false},
		{"UTF-8", false},
		{"SHA-256", false},
		{"2026-11-24", false},
		{"internationalization", false},
		{"my-device-name-here", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			if got := isBareCode(tc.in); got != tc.want {
				t.Errorf("isBareCode(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
