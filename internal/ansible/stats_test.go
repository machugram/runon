package ansible

import "testing"

func TestParseRecap(t *testing.T) {
	raw := []byte("PLAY RECAP *********************************************************************\n" +
		"\x1b[0;32mdev-app-1\x1b[0m                  : ok=21   changed=0    unreachable=0    failed=0    skipped=1\n" +
		"dev-proxy                  : ok=14   changed=0    unreachable=0    failed=0    skipped=2\n")
	stats, err := ParseRecap(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := AssertClean(stats); err != nil {
		t.Fatal(err)
	}
}

func TestParseRecapRejectsChanges(t *testing.T) {
	raw := []byte("dev-app-1                  : ok=4    changed=2    unreachable=0    failed=0    skipped=0\n" +
		"dev-proxy                  : ok=4    changed=0    unreachable=1    failed=0    skipped=0\n")
	stats, err := ParseRecap(raw)
	if err != nil {
		t.Fatal(err)
	}
	err = AssertClean(stats)
	if err == nil {
		t.Fatal("expected changed hosts to fail")
	}
	if err.Error() != "playbook was not idempotent: dev-app-1 changed=2; dev-proxy failures=0 unreachable=1" {
		t.Fatalf("error: %s", err)
	}
}

func TestParseRecapMissing(t *testing.T) {
	if _, err := ParseRecap([]byte("no recap\n")); err == nil {
		t.Fatal("expected missing recap to fail")
	}
}
