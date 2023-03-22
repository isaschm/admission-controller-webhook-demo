package transparency

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

const (
	UnspecifiedTag = "unspecified"
)

type Tags struct {
	Purposes           string `mapstructure:"purposes"`
	LegitimateInterest string `mapstructure:"purposes"`
	LegalBasis         string `mapstructure:"legalBasis"`
}

func DecodeTags(m map[string]string) (*Tags, error) {
	for _, key := range m {
		if m[key] == UnspecifiedTag {
			m[key] = ""
		}
	}

	tags := new(Tags)
	if err := mapstructure.Decode(m, tags); err != nil {
		return nil, fmt.Errorf("decoding map to Tags: %w", err)
	}

	return tags, nil
}

func (t *Tags) Encode() (map[string]string, error) {
	tag := *t

	if tag.Purposes == "" {
		tag.Purposes = UnspecifiedTag
	}
	if tag.LegitimateInterest == "" {
		tag.LegitimateInterest = UnspecifiedTag
	}
	if tag.LegalBasis == "" {
		tag.LegalBasis = UnspecifiedTag
	}

	encoded := make(map[string]string)
	if err := mapstructure.Decode(tag, encoded); err != nil {
		return nil, fmt.Errorf("encoding tag struct to map: %w", err)
	}

	return encoded, nil
}
