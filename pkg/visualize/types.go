package visualize

import (
	"fmt"
	"github.com/alcideio/rbac-tool/pkg/rbac"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Opts struct {
	//Input source - cluster or input file/stdin
	ClusterContext string
	Infile         string

	//Show Actuall use by Pods
	ShowPodsOnly bool

	Outfile    string
	Outformat  string
	ShowRules  bool
	ShowLegend bool

	IncludedNamespaces string
	ExcludedNamespaces string

	IncludeSubjectsRegex string
}

func (o *Opts) Validate() error {
	if o.Infile != "" && o.ClusterContext != "" {
		return fmt.Errorf("Either use input file or specify cluster context")
	}

	return nil
}

type Permissions struct {
	rbac.Permissions

	ServiceAccountsUsed sets.String
	Pods                map[string]map[string]v1.Pod //map[namespace]map[name]Pod
}
