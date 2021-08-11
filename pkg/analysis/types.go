package analysis

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"sigs.k8s.io/yaml"
)

const (
	SEVERITY_CRIT = "CRITICAL"
	SEVERITY_HIGH = "HIGH"
	SEVERITY_MED = "MEDIUM"
	SEVERITY_INFO = "INFO"
)

//Analysis Rule
type Rule struct {
	//Rule Name
	Name string
	//Rule Description
	Description string
	//Rule Recommendation - rendered as a Google CEL expression to customize the message
	Recommendation string
	//Rule UUID
	Uuid string
	//Rule UUID
	Severity string

	//Documetation & additional reading references
	References []string

	//A Google CEL expression analysis rule.
	// Input: []SubjectPolicyList
	// Output: Boolean
	AnalysisExpr string

	//Any Resources that we should not report about.
	// For example do not report on findings from kube-system namespace
	Exclusions []Exclusion
}

type Exclusion struct {
	//Is this exclusion turned off
	Disabled bool

	//Exclusion note
	Comment string

	//Who added this exclusion
	AddedBy string

	//When this exclusion had changed -
	LastModified string

	//Snooze this exception until X - time since epoch
	SnoozeUntil uint64

	//A Google CEL expression exceptions
	// Input: v1.Subject
	// Output: Boolean
	Expression string
}

type Rules []Rule

type AnalysisConfigInfo struct {
	//Config Name
	Name string
	//Rule Description
	Description string
	//Rule UUID
	Uuid string
}

type AnalysisConfig struct {
	AnalysisConfigInfo

	Rules []Rule
}

func ExportAnalysisConfig(format string, c *AnalysisConfig) (string, error) {
	switch format {
	case "yaml":

		data, err := yaml.Marshal(c)
		if err != nil {
			return "", fmt.Errorf("Processing error - %v", err)
		}

		return string(data), nil

	case "json":
		data, err := json.Marshal(c)
		if err != nil {
			return "", fmt.Errorf("Processing error - %v", err)
		}

		return string(data), nil

	default:
		return "", fmt.Errorf("Unsupported output format")
	}
}

func LoadAnalysisConfig(fname string) (*AnalysisConfig, error) {
	c := &AnalysisConfig{}

	yamlFile, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
