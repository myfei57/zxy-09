package coord

import (
	"errors"
	"fmt"
)

// Join adds an intersection to a coordinated group and recalculates the
// group's offsets against the current member list.
func (c *Coordinator) Join(groupID, intersectionID string) (Group, error) {
	g, err := c.Get(groupID)
	if err != nil {
		return Group{}, err
	}
	for _, member := range g.Members {
		if member == intersectionID {
			return Group{}, fmt.Errorf("intersection %s is already in group %s", intersectionID, groupID)
		}
	}
	snapshot := g
	g.Members = append(g.Members, intersectionID)
	g.UpdatedAt = c.now()
	if err := c.fs.WriteJSON(c.Path(g.ID), g); err != nil {
		return Group{}, err
	}
	if err := c.Recalculate(snapshot); err != nil {
		return Group{}, err
	}
	if c.audit != nil {
		_, _ = c.audit.Record("console", "coord.join", "group", g.ID, fmt.Sprintf("member %s joined %s", intersectionID, g.Name))
	}
	return g, nil
}

// Leave removes an intersection from a group and recalculates offsets for the
// remaining members.
func (c *Coordinator) Leave(groupID, intersectionID string) (Group, error) {
	g, err := c.Get(groupID)
	if err != nil {
		return Group{}, err
	}
	kept := g.Members[:0]
	found := false
	for _, member := range g.Members {
		if member == intersectionID {
			found = true
			continue
		}
		kept = append(kept, member)
	}
	if !found {
		return Group{}, errors.New("intersection is not a group member")
	}
	g.Members = kept
	g.UpdatedAt = c.now()
	if err := c.fs.WriteJSON(c.Path(g.ID), g); err != nil {
		return Group{}, err
	}
	if err := c.Recalculate(g); err != nil {
		return Group{}, err
	}
	return g, nil
}
