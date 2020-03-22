![Go](https://github.com/alcideio/rbac-minimize/workflows/Go/badge.svg)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

# Kubernetes RBAC 

Role-based access control (RBAC) is a method of regulating access to computer or network resources based on the roles of individual users within your organization.
RBAC authorization uses the `rbac.authorization.k8s.io` API group to drive authorization decisions, allowing you to dynamically configure policies through the Kubernetes API.

Permissions are purely **additive** (there are **no “deny”** rules).

A Role always sets permissions within a particular namespace ; when you create a Role, you have to specify the namespace it belongs in.
ClusterRole, by contrast, is a non-namespaced resource.
ClusterRoles have several uses. You can use a ClusterRole to:

- define permissions on namespaced resources and be granted within individual namespace(s)
- define permissions on namespaced resources and be granted across all namespaces
- define permissions on cluster-scoped resources

If you want to define a role within a namespace, use a Role; if you want to define a role cluster-wide, use a ClusterRole.

**rbac-minimize** simplifies the creation process of RBAC policies and avoiding those wildcards `*` and it adapts to specific Kubernets API server

# Say Hello to `rbac-minimize`

Generate Role or ClusterRole resource while reducing the use of wildcards.

`rbac-minimize` reads from the Kubernetes discovery API the available API Groups and resources, 
and based on the command line options, generate an explicit Role/ClusterRole that avoid wildcards.

One simple example is to create a Role/ClusterRole that can read everything except `secrets` 

####  Running

```bash
rbac-minimize  gen --generated-type=Role --deny-resources=secrets.,ingresses.extensions --allowed-verbs=get,list --allowed-groups=,apps,networking.k8s.io
```

#### Would yield:
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

## Examples:

- Generate a Role with read-only (get,list) excluding secrets (core group) and ingresses (extensions group) 
```shell script
rbac-minimize gen --generated-type Role --deny-resources=secrets.,ingresses.extensions --allowed-verbs=get,list
```


- Generate a Role with read-only (get,list) excluding secrets (core group) from core group, admissionregistration.k8s.io,storage.k8s.io,networking.k8s.io
```shell script
rbac-minimize gen --generated-type ClusterRole --deny-resources=secrets., --allowed-verbs=get,list  --allowed-groups=,admissionregistration.k8s.io,storage.k8s.io,networking.k8s.io
```


## Command Line Reference

```bash
Generate Role or ClusterRole resource while reducing the use of wildcards.

rbac-minimize read from the Kubernetes discovery API the available API Groups and resources, 
and based on the command line options, generate an explicit Role/ClusterRole that avoid wildcards

Examples:

# Generate a Role with read-only (get,list) excluding secrets (core group) and ingresses (extensions group) 
rbac-minimize gen --generated-type=Role --deny-resources=secrets.,ingresses.extensions --allowed-verbs=get,list

# Generate a Role with read-only (get,list) excluding secrets (core group) from core group, admissionregistration.k8s.io,storage.k8s.io,networking.k8s.io
rbac-minimize gen --generated-type=ClusterRole --deny-resources=secrets., --allowed-verbs=get,list  --allowed-groups=,admissionregistration.k8s.io,storage.k8s.io,networking.k8s.io

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

