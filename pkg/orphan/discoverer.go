package Orphan

import (
	"fmt"
	"github.com/alcideio/rbac-tool/pkg/kube"
	"github.com/alcideio/rbac-tool/pkg/rbac"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Discoverer struct {
	perms  *rbac.Permissions
	client *kube.KubeClient

	saInUse sets.String // Namespace/ServiceAccountName
}

func NewDiscoverer(client *kube.KubeClient) (*Discoverer, error) {
	perms, err := rbac.NewPermissionsFromCluster(client)

	if err != nil {
		return nil, err
	}

	saInUse, err := client.ListUsedServiceAccounts()
	if err != nil {
		return nil, err
	}

	od := &Discoverer{
		perms:   perms,
		client:  client,
		saInUse: saInUse,
	}

	return od, nil
}

func (od *Discoverer) GetOrphanServiceAccounts() ([]*v1.ServiceAccount, error) {
	var orphanList []*v1.ServiceAccount

	for nsName, sas := range od.perms.ServiceAccounts {
		for saName, sa := range sas {
			if !od.saInUse.Has(fmt.Sprintf("%s/%s", nsName, saName)) {
				serviceAccount := sa.DeepCopy()
				orphanList = append(orphanList, serviceAccount)
			}
		}
	}

	return orphanList, nil
}
