package shell

import "testing"

func TestUniquePreserve(t *testing.T) {
	got := uniquePreserve([]string{"a", "b", "a", "", "c"})
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("%v", got)
	}
}

func TestCollectIncomingNoWait(t *testing.T) {
	got := CollectIncoming([]string{"a", "a", "b"}, 0)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("%v", got)
	}
}
