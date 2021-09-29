## <img src="https://www.rapid7.com/Areas/Docs/includes/img/r7-nav/Rapid7_logo-short.svg" alt="insightCloudSec" width="28"/> | insightCloudSec | RBAC TOOL  

A collection of Kubernetes RBAC tools to sugar coat Kubernetes RBAC complexity

## Install

#### Standalone

```shell script
curl https://raw.githubusercontent.com/alcideio/rbac-tool/master/download.sh | bash
```

#### kubectl plugin // <img src="https://raw.githubusercontent.com/kubernetes-sigs/krew/master/assets/logo/horizontal/color/krew-horizontal-color.png" alt="krew" width="48"/> //  

```shell script
$ kubectl krew install rbac-tool
```

## Command Line Examples (Standalone)

```shell script
# Show which users/groups/service accounts are allowed to read secrets in the cluster pointed by kubeconfig
rbac-tool who-can get secrets

# Scan the cluster pointed by the kubeconfig context 'myctx'
rbac-tool viz --cluster-context myctx

# Scan and create a PNG image from the graph
rbac-tool viz --outformat dot --exclude-namespaces=soemns && cat rbac.dot | dot -Tpng > rbac.png && google-chrome rbac.png
# Render Online
https://dreampuf.github.io/GraphvizOnline

# Analyze cluster RBAC permissions to identify overly permissive roles and principals
rbac-tool analysis -o table

# Search All Service Accounts That Contains myname
rbac-tool lookup -e '.*myname.*'

# Lookup all accounts that DO NOT start with system: )
rbac-tool lookup -ne '^system:.*'

# List policy rules for users (or all of them)
rbac-tool policy-rules -e '^system:anonymous'

# Generate from Audit events & Visualize 
rbac-tool auditgen -f testdata  | rbac-tool viz   -f -

# Generate a `ClusterRole` policy that allows to read everything **except** *secrets* and *services*
rbac-tool  gen  --deny-resources=secrets.,services. --allowed-verbs=get,list
```

## kubectl rbac-tool ...

```shell script
# Generate HTML visualzation of your RBAC permissions
kubectl rbac-tool viz

# Query who can read secrets
kubectl rbac-tool who-can get secret

# Generate a ClusterRole policy that allows to read everything except secrets and services
kubectl rbac-tool gen --deny-resources=secrets.,services. --allowed-verbs=get,list

# Analyze cluster RBAC permissions to identify overly permissive roles and principals
kubectl rbac-tool analysis -o table

```
