package entitlements

import (
	"iter"
	"slices"

	"github.com/FacileStudio/porte/directory"
)

// Known reports whether the subject appeared in the last directory snapshot,
// as a person with entitlements or as a member of a space. Before any
// snapshot it is the people the restored state holds values for.
func (p *Plans) Known(subject string) bool {
	set := p.subjects.Load()
	if set == nil || subject == "" {
		return false
	}
	_, ok := (*set)[subject]
	return ok
}

// learn remembers every subject the snapshot names, so a user holding several
// identities under the issuer is read under the one Registre knows today.
func (p *Plans) learn(snap directory.Snapshot) {
	subjects := make([]string, 0, len(snap.People))
	for _, person := range snap.People {
		subjects = append(subjects, person.Subject)
	}
	for _, space := range snap.Spaces {
		for _, member := range space.Members {
			subjects = append(subjects, member.Subject)
		}
	}
	p.remember(slices.Values(subjects))
}

// remember replaces the known subjects with the given ones.
func (p *Plans) remember(subjects iter.Seq[string]) {
	set := map[string]struct{}{}
	for subject := range subjects {
		if subject != "" {
			set[subject] = struct{}{}
		}
	}
	p.subjects.Store(&set)
}
