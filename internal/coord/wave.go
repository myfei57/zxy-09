package coord

import "fmt"

// WaveReport describes the health of a coordinated green wave.
type WaveReport struct {
	GroupID        string
	MemberCount    int
	OffsetCoverage int
	Missing        []string
	Consistent     bool
}

// ValidateGreenWave checks that every member has an offset and that the
// offsets are distinct enough to form a wave rather than colliding greens.
func (c *Coordinator) ValidateGreenWave(groupID string) (WaveReport, error) {
	g, err := c.Get(groupID)
	if err != nil {
		return WaveReport{}, err
	}
	report := WaveReport{
		GroupID:        g.ID,
		MemberCount:    len(g.Members),
		OffsetCoverage: len(g.Offsets),
	}
	seen := make(map[int]string)
	for _, member := range g.Members {
		offset, ok := g.Offsets[member]
		if !ok {
			report.Missing = append(report.Missing, member)
			continue
		}
		if other, exists := seen[offset]; exists {
			report.Consistent = false
			return report, fmt.Errorf("members %s and %s collide at offset %d", other, member, offset)
		}
		seen[offset] = member
	}
	report.Consistent = len(report.Missing) == 0 && len(seen) == len(g.Members)
	return report, nil
}
