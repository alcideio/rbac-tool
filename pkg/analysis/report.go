package analysis

import v1 "k8s.io/api/rbac/v1"

type AnalysisReport struct {
	//The Analysis Config Info
	AnalysisConfigInfo AnalysisConfigInfo

	Stats AnalysisStats

	//Report Create Time
	CreatedOn string

	Findings []AnalysisReportFinding
}

type AnalysisStats struct {
	//Analysis Rules
	RuleCount int
}

type AnalysisReportFinding struct {
	Subject *v1.Subject

	Finding AnalysisFinding
}

type AnalysisFinding struct {
	// Finding Severity
	Severity string

	//Rule Name
	Message string

	//Rule Description
	Recommendation string

	//The Rule Name that triggered this finding
	RuleName string
	//The Rule UUID that triggered this finding
	RuleUuid string

	//Documetation & additional reading references
	References []string
}

