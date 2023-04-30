package transparency

const (
	UnspecifiedTag = "unspecified"
)

var (
	transparencyTags = []string{"dataCategories", "necessity", "autoDecision"}
)

func DecodeTags(m map[string]string) (map[string]string, error) {
	for _, tag := range transparencyTags {
		if m[tag] == "" {
			m[tag] = UnspecifiedTag
		}
	}

	return m, nil
}
