package analysis

import (
	_ "embed"

	"sigs.k8s.io/yaml"
)

//go:embed default-rules.yaml
var defaultAnalysis []byte

func DefaultAnalysisConfig() *AnalysisConfig {
	c := AnalysisConfig{}

	if err := yaml.Unmarshal(defaultAnalysis, &c); err != nil {
		return nil
	}

	return &c
}

func ExportDefaultConfig(format string) (string, error) {
	c := DefaultAnalysisConfig()

	return ExportAnalysisConfig(format, c)
}
