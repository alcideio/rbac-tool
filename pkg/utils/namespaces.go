package utils

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"strings"
)

func GetNamespaceSets(nsInclude string, nsExclude string) (sets.String, sets.String) {
	NamespaceInclude := sets.NewString()
	NamespaceExclude := sets.NewString()

	if nsInclude == "" {
		NamespaceInclude.Insert("*")
	} else {
		inclusionList := strings.TrimSpace(nsInclude)
		inclusion := strings.Split(inclusionList, ",")

		for _, inc := range inclusion {
			NamespaceInclude.Insert(strings.ToLower(inc))
		}
	}

	if nsExclude != "" {
		exclusionList := strings.TrimSpace(nsExclude)
		exclusion := strings.Split(exclusionList, ",")

		for _, exc := range exclusion {
			NamespaceExclude.Insert(strings.ToLower(exc))
		}
	}

	return NamespaceInclude, NamespaceExclude
}

func IsNamespaceIncluded(namespace string, nsInclude sets.String, nsExclude sets.String) bool {

	if nsExclude.Len() > 0 && isInNamespace(namespace, nsExclude) {
		return false
	}

	if nsInclude.Len() > 0 && isInNamespace(namespace, nsInclude) {
		return true
	}

	return false
}

func isInNamespace(namespace string, namespaceSet sets.String) (valid bool) {
	if namespaceSet.Has(v1.NamespaceAll) || namespaceSet.Has("*") || namespaceSet.Has(strings.ToLower(namespace)) {
		return true
	}

	return false
}
