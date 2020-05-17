# rbac-tool

<img src="https://github.com/alcideio/rbac-tool/raw/master/rbac-tool.png" alt="rbac-tool" width="128"/>

A collection of Kubernetes RBAC tools to sugar coat Kubernetes RBAC complexity

## Install

```shell script
curl https://raw.githubusercontent.com/alcideio/rbac-tool/master/download.sh | bash
```

## Command Line Examples

```shell script
# Scan the cluster pointed by the kubeconfig context 'myctx'
rbac-tool viz --cluster-context myctx

# Scan and create a PNG image from the graph
rbac-tool viz --outformat dot --exclude-namespaces=soemns && cat rbac.dot | dot -Tpng > rbac.png && google-chrome rbac.png
# Render Online
https://dreampuf.github.io/GraphvizOnline

# Search All Service Accounts That Contains myname
rbac-tool lookup -e '.*myname.*'

# Lookup all accounts that DO NOT start with system: )
rbac-tool lookup -ne '^system:.*'


# Generate a `ClusterRole` policy that allows to read everything **except** *secrets* and *services*
rbac-tool  gen  --deny-resources=secrets.,services. --allowed-verbs=get,list
```