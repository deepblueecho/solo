package service

type wakeRouteInput struct {
	ActiveIDs    []string
	MentionedIDs []string
	HasMention   bool
	Edges        []relationshipEdge
}

func selectWakeAgentIDs(in wakeRouteInput) []string {
	if len(in.MentionedIDs) > 0 {
		return activeMentionedIDs(in.ActiveIDs, in.MentionedIDs)
	}
	if in.HasMention {
		return nil
	}
	return selectCoordinatorID(in.ActiveIDs, in.Edges)
}

func activeMentionedIDs(activeIDs, mentionedIDs []string) []string {
	mentioned := make(map[string]bool, len(mentionedIDs))
	for _, id := range mentionedIDs {
		mentioned[id] = true
	}
	out := []string{}
	for _, id := range activeIDs {
		if mentioned[id] {
			out = append(out, id)
		}
	}
	return out
}

func selectCoordinatorID(activeIDs []string, edges []relationshipEdge) []string {
	if len(activeIDs) <= 1 {
		return append([]string(nil), activeIDs...)
	}
	active := make(map[string]bool, len(activeIDs))
	for _, id := range activeIDs {
		active[id] = true
	}
	hasParent := map[string]bool{}
	for _, edge := range edges {
		if active[edge.from] && active[edge.to] {
			hasParent[edge.to] = true
		}
	}
	for _, id := range activeIDs {
		if !hasParent[id] {
			return []string{id}
		}
	}
	return []string{activeIDs[0]}
}
