package notespec

import "testing"

func TestByRef(t *testing.T) {
	s, ok := ByRef("n-obj-ab12")
	if !ok || s.ID != "objects-as-they-show-up-in-a-request" {
		t.Fatalf("got %+v ok=%v", s, ok)
	}
	s, ok = ByRef("note-computer-networks-1")
	if !ok || s.ID != "computer-networks" {
		t.Fatalf("legacy got %+v ok=%v", s, ok)
	}
}

func TestByHint(t *testing.T) {
	s, ok := ByHint("objects-as-they-show-up-in-a-request", "", "")
	if !ok || s.AmountPaise != 4900 {
		t.Fatalf("got %+v ok=%v", s, ok)
	}
	s, ok = ByHint("", "Objects as they show up in a request", "")
	if !ok || s.ID != "objects-as-they-show-up-in-a-request" {
		t.Fatalf("legacy title got %+v ok=%v", s, ok)
	}
	s, ok = ByHint("", "Low-level design, as it shows up in a request", "")
	if !ok || s.ID != "objects-as-they-show-up-in-a-request" {
		t.Fatalf("new title got %+v ok=%v", s, ok)
	}
	s, ok = ByHint("", "Computer networks, as they show up on a box", "")
	if !ok || s.ID != "computer-networks" {
		t.Fatalf("desc got %+v ok=%v", s, ok)
	}
}
