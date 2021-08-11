module github.com/alcideio/rbac-tool

go 1.16

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/antonmedv/expr v1.8.9
	github.com/emicklei/dot v0.10.2
	github.com/fatih/color v1.7.0
	github.com/fatih/structs v1.1.0
	github.com/google/cel-go v0.7.3
	github.com/google/uuid v1.1.2
	github.com/huandu/xstrings v1.3.0 // indirect
	github.com/kr/pretty v0.2.0
	github.com/kylelemons/godebug v0.0.0-20170820004349-d65d576e9348
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/olekukonko/tablewriter v0.0.0-20170122224234-a0225b3f23b5
	github.com/spf13/cobra v1.0.0
	google.golang.org/protobuf v1.25.0
	k8s.io/api v0.19.13
	k8s.io/apimachinery v0.19.13
	k8s.io/apiserver v0.19.13
	k8s.io/client-go v0.19.13
	k8s.io/klog v1.0.0
	k8s.io/kubernetes v1.19.13
	sigs.k8s.io/yaml v1.2.0
)

replace (
	//
	// k8s.io/kubernetes this is evil....but nessecary for audit2rbac
	//
	k8s.io/api => k8s.io/api v0.19.13
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.13
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.13
	k8s.io/apiserver => k8s.io/apiserver v0.19.13
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.13
	k8s.io/client-go => k8s.io/client-go v0.19.13
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.13
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.13
	k8s.io/code-generator => k8s.io/code-generator v0.19.13
	k8s.io/component-base => k8s.io/component-base v0.19.13
	k8s.io/cri-api => k8s.io/cri-api v0.19.13
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.13
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.13
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.13
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.13
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.13
	k8s.io/kubectl => k8s.io/kubectl v0.19.13
	k8s.io/kubelet => k8s.io/kubelet v0.19.13
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.13
	k8s.io/metrics => k8s.io/metrics v0.19.13
	k8s.io/node-api => k8s.io/node-api v0.19.13
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.13
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.19.13
	k8s.io/sample-controller => k8s.io/sample-controller v0.19.13
)
