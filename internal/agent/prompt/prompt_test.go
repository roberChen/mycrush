package prompt

import (
	"testing"

	"github.com/charmbracelet/crush/internal/skills"
	"github.com/stretchr/testify/require"
)

func newSkillNames(names ...string) []*skills.Skill {
	out := make([]*skills.Skill, 0, len(names))
	for _, n := range names {
		out = append(out, &skills.Skill{Name: n})
	}
	return out
}

func skillNames(in []*skills.Skill) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, s.Name)
	}
	return out
}

func TestFilterSkillsByName(t *testing.T) {
	t.Parallel()

	sk := newSkillNames("a", "b", "c", "d")
	got := filterSkillsByName(sk, []string{"b", "d", "missing"})
	require.Equal(t, []string{"b", "d"}, skillNames(got))
}

func TestWithAllowedSkills_OptionApplies(t *testing.T) {
	t.Parallel()

	p := &Prompt{}
	WithAllowedSkills([]string{"x", "y"})(p)
	require.Equal(t, []string{"x", "y"}, p.allowedSkills)

	// Empty slice resets to nil-equivalent (no restriction).
	WithAllowedSkills(nil)(p)
	require.Nil(t, p.allowedSkills)
}

