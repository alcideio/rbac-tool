![Go](https://github.com/gadinaor/rbac-minimize/workflows/Go/badge.svg)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

# rbac-minimize

Generate Role or ClusterRole resource while reducing the use of wildcards.

rbac-generator read from the Kubernetes discovery API the available API Groups and resources, 
and based on the command line options, generate an explicit Role/ClusterRole that avoid wildcards

Running
```shell script
rbac-minimizer  gen --generated-type=Role --deny-resources=secrets.,ingresses.extensions --allowed-verbs=get,list --allowed-groups=,apps,networking.k8s.io
```

Would yield:
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: custom-role
  namespace: mynamespace
rules:
- apiGroups:
  - ""
  resources:
  - resourcequotas
  - pods
  - bindings
  - replicationcontrollers
  - podtemplates
  - services
  - limitranges
  - serviceaccounts
  - configmaps
  - events
  - componentstatuses
  - namespaces
  - endpoints
  - nodes
  - persistentvolumes
  - persistentvolumeclaims
  verbs:
  - get
  - list
- apiGroups:
  - apps
  resources:
  - '*'
  verbs:
  - get
  - list
- apiGroups:
  - networking.k8s.io
  resources:
  - '*'
  verbs:
  - get
  - list

```

###Examples:

- Generate a Role with read-only (get,list) excluding secrets (core group) and ingresses (extensions group) 
```shell script
rbac-minimizer gen --generated-type Role --deny-resources=secrets.,ingresses.extensions --allowed-verbs=get,list
```


- Generate a Role with read-only (get,list) excluding secrets (core group) from core group, admissionregistration.k8s.io,storage.k8s.io,networking.k8s.io
```shell script
rbac-minimizer gen --generated-type ClusterRole --deny-resources=secrets., --allowed-verbs=get,list  --allowed-groups=,admissionregistration.k8s.io,storage.k8s.io,networking.k8s.io
```


## Command Line Reference

```shell script
Usage:
  rbac-minimize generate [flags]

Aliases:
  generate, gen

Flags:
      --allowed-groups strings   Comma separated list of API groups we would like to allow '*' (default [*])
      --allowed-verbs strings    Comma separated list of verbs to include. To include all use '* (default [*])
  -c, --cluster-context string   Cluster.use 'kubectl config get-contexts' to list available contexts
      --deny-resources strings   Comma separated list of resource.group
  -t, --generated-type string    Role or ClusteRole (default "ClusterRole")
  -h, --help                     help for generate


```

