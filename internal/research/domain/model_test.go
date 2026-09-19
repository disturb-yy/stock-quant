package domain

import "testing"

func TestNormalizeResearchProjectInput(t *testing.T) {
	description := "  观察估值与行业变化  "
	normalized, err := NormalizeResearchProjectInput(ResearchProjectInput{Name: "  银行研究  ", Description: &description})
	if err != nil {
		t.Fatalf("NormalizeResearchProjectInput() error = %v", err)
	}
	if normalized.Name != "银行研究" || normalized.Description == nil || *normalized.Description != "观察估值与行业变化" {
		t.Fatalf("normalized input = %#v, want trimmed fields", normalized)
	}
}

func TestNormalizeResearchProjectInputRejectsInvalidFields(t *testing.T) {
	longDescription := string(make([]rune, MaxResearchDescriptionLength+1))
	tests := []struct {
		name  string
		input ResearchProjectInput
		field string
	}{
		{name: "missing name", input: ResearchProjectInput{}, field: "name"},
		{name: "long name", input: ResearchProjectInput{Name: string(make([]rune, MaxResearchNameLength+1))}, field: "name"},
		{name: "long description", input: ResearchProjectInput{Name: "项目", Description: &longDescription}, field: "description"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NormalizeResearchProjectInput(test.input)
			if err == nil {
				t.Fatal("NormalizeResearchProjectInput() error = nil")
			}
			validation, ok := err.(*ValidationError)
			if !ok || validation.Fields[test.field] == "" {
				t.Fatalf("error = %#v, want field %q", err, test.field)
			}
		})
	}
}
