package profile

import "testing"

func TestNormalizeUsername(t *testing.T) {
	valid := map[string]string{"@Teste2": "teste2", " ana.silva ": "ana.silva", "joao_99": "joao_99"}
	for raw, want := range valid {
		if got, ok := NormalizeUsername(raw); !ok || got != want {
			t.Fatalf("NormalizeUsername(%q) = %q, %v", raw, got, ok)
		}
	}
	for _, raw := range []string{"", "ab", ".ana", "ana..silva", "ana.", "ana silva", "ana-silva", "ãna", "averyveryveryverylongusername_31x"} {
		if got, ok := NormalizeUsername(raw); ok {
			t.Fatalf("NormalizeUsername(%q) accepted %q", raw, got)
		}
	}
	if !IsReserved("admin") || IsReserved("teste2") {
		t.Fatal("reserved names")
	}
}

func TestNormalizeDisplayName(t *testing.T) {
	if got, ok := NormalizeDisplayName("  Edgar   Silva "); !ok || got != "Edgar Silva" {
		t.Fatalf("got %q %v", got, ok)
	}
	if got, ok := NormalizeDisplayName("   "); !ok || got != "" {
		t.Fatalf("blank clears: %q %v", got, ok)
	}
	for _, raw := range []string{"<script>", "a\u0000b", "a very long display name that exceeds forty characters"} {
		if _, ok := NormalizeDisplayName(raw); ok {
			t.Fatalf("NormalizeDisplayName(%q) accepted", raw)
		}
	}
}
