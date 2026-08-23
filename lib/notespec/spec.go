package notespec

import "strings"

type Spec struct {
	ID          string
	Title       string
	Filename    string
	RefPrefix   string
	AmountPaise int
}

var catalog = []Spec{
	{
		ID:          "computer-networks",
		Title:       "Computer networks, as they show up on a box",
		Filename:    "computer-networks.pdf",
		RefPrefix:   "n-cn-",
		AmountPaise: 4900,
	},
	{
		ID:          "objects-as-they-show-up-in-a-request",
		Title:       "Low-level design, as it shows up in a request",
		Filename:    "objects-as-they-show-up-in-a-request.pdf",
		RefPrefix:   "n-obj-",
		AmountPaise: 4900,
	},
}

func All() []Spec {
	return catalog
}

func ByID(id string) (Spec, bool) {
	id = strings.TrimSpace(id)
	for _, s := range catalog {
		if s.ID == id {
			return s, true
		}
	}
	return Spec{}, false
}

func ByRef(ref string) (Spec, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return Spec{}, false
	}
	for _, s := range catalog {
		if strings.HasPrefix(ref, s.RefPrefix) {
			return s, true
		}
	}
	// Older payment links used this reference.
	if strings.HasPrefix(ref, "note-computer-networks") {
		return catalog[0], true
	}
	return Spec{}, false
}

func ByHint(noteID, description, ref string) (Spec, bool) {
	if s, ok := ByID(noteID); ok {
		return s, true
	}
	if s, ok := ByRef(ref); ok {
		return s, true
	}
	d := strings.ToLower(description)
	for _, s := range catalog {
		if d != "" && strings.Contains(d, strings.ToLower(s.Title)) {
			return s, true
		}
	}
	if strings.Contains(d, "objects as they show up") {
		if s, ok := ByID("objects-as-they-show-up-in-a-request"); ok {
			return s, true
		}
	}
	return Spec{}, false
}
