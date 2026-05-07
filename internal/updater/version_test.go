package updater

import "testing"

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v0.5.0", "v0.4.0", 1},
		{"v0.4.0", "v0.5.0", -1},
		{"v0.5.0", "v0.5.0", 0},
		{"0.5.0", "v0.5.0", 0},
		{"v0.5.0", "0.5.0", 0},
		{"v0.5.0-rc1", "v0.5.0", -1},
		{"v0.5.0", "v0.5.0-rc1", 1},
		{"v0.5.0-rc2", "v0.5.0-rc1", 1},
		{"v1.0.0", "v0.99.0", 1},
		{"", "", 0},
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestCompareInvalid(t *testing.T) {
	if got := Compare("not-a-version", "v0.5.0"); got != -1 {
		t.Fatalf("invalid a should sort below valid b: got %d", got)
	}
	if got := Compare("v0.5.0", "not-a-version"); got != 1 {
		t.Fatalf("valid a should sort above invalid b: got %d", got)
	}
	if got := Compare("not-a-version", "also-not"); got != 0 {
		t.Fatalf("two invalids should be equal: got %d", got)
	}
}

func TestIsValidVersion(t *testing.T) {
	if !IsValidVersion("v0.5.0") {
		t.Fatal("v0.5.0 should be valid")
	}
	if !IsValidVersion("0.5.0") {
		t.Fatal("0.5.0 should be valid (auto-prefixed)")
	}
	if IsValidVersion("not-a-version") {
		t.Fatal("nonsense should be invalid")
	}
	if IsValidVersion("") {
		t.Fatal("empty should be invalid")
	}
}
