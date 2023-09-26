package audit

import (
	"fmt"
	"io"

	"net/url"
	"reflect"
	"sort"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/apis/audit"
	auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
	"k8s.io/apiserver/pkg/authentication/serviceaccount"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	validation_helper "k8s.io/component-helpers/auth/rbac/validation"
	rbacv1helper "k8s.io/kubernetes/pkg/apis/rbac/v1"
	"k8s.io/kubernetes/pkg/registry/rbac/validation"
)

func userToSubject(user user.Info) rbacv1.Subject {
	if ns, name, err := serviceaccount.SplitUsername(user.GetName()); err == nil {
		return rbacv1.Subject{Name: name, Namespace: ns, Kind: "ServiceAccount"}
	}
	return rbacv1.Subject{Name: user.GetName(), Kind: "User", APIGroup: rbacv1.GroupName}
}

func attributesToResourceRule(request authorizer.AttributesRecord, options GenerateOptions) rbacv1.PolicyRule {
	verbs := append([]string{request.Verb}, options.VerbExpansions[request.Verb]...)
	rule := rbacv1helper.NewRule(verbs...).Groups(request.APIGroup).Resources(request.Resource).RuleOrDie()
	if request.Subresource != "" {
		rule.Resources[0] = rule.Resources[0] + "/" + request.Subresource
	}
	if request.Name != "" {
		rule.ResourceNames = []string{request.Name}
	}
	return rule
}

func compactRules(rules []rbacv1.PolicyRule) []rbacv1.PolicyRule {
	breakdownRules := []rbacv1.PolicyRule{}
	for _, rule := range rules {
		breakdownRules = append(breakdownRules, validation_helper.BreakdownRule(rule)...)
	}
	compactRules, err := validation.CompactRules(breakdownRules)
	if err != nil {
		return rules
	}
	// TODO: fix CompactRules to dedupe verbs
	for i := range compactRules {
		compactRules[i].Verbs = sets.NewString(compactRules[i].Verbs...).List()
	}

	accumulatingRules := []rbacv1.PolicyRule{}
	for _, rule := range compactRules {
		// Non-resource rules just accumulate
		if len(rule.Resources) == 0 {
			accumulatingRules = append(accumulatingRules, rule)
			continue
		}

		accumulated := false
		// strip resource
		resourcelessRule := rule
		resourcelessRule.Resources = nil
		// strip name
		namelessRule := rule
		namelessRule.ResourceNames = nil
		for j, accumulatingRule := range accumulatingRules {
			// strip name
			namelessAccumulatingRule := accumulatingRule
			namelessAccumulatingRule.ResourceNames = nil
			if reflect.DeepEqual(namelessRule, namelessAccumulatingRule) {
				combinedNames := sets.NewString(accumulatingRule.ResourceNames...)
				combinedNames.Insert(rule.ResourceNames...)
				accumulatingRule.ResourceNames = combinedNames.List()
				accumulatingRules[j] = accumulatingRule
				accumulated = true
				break
			}

			// strip resource
			resourcelessAccumulatingRule := accumulatingRule
			resourcelessAccumulatingRule.Resources = nil
			if reflect.DeepEqual(resourcelessRule, resourcelessAccumulatingRule) {
				combinedResources := sets.NewString(accumulatingRule.Resources...)
				combinedResources.Insert(rule.Resources...)
				accumulatingRule.Resources = combinedResources.List()
				accumulatingRules[j] = accumulatingRule
				accumulated = true
				break
			}
		}
		if !accumulated {
			accumulatingRules = append(accumulatingRules, rule)
		}
	}

	sort.SliceStable(accumulatingRules, func(i, j int) bool {
		// TODO: fix upstream sorting to prioritize API group
		if c := strings.Compare(strings.Join(accumulatingRules[i].APIGroups, ","), strings.Join(accumulatingRules[j].APIGroups, ",")); c != 0 {
			return c < 0
		}
		return strings.Compare(rbacv1helper.CompactString(accumulatingRules[i]), rbacv1helper.CompactString(accumulatingRules[j])) < 0
	})
	return accumulatingRules
}

func sortRequests(requests []authorizer.AttributesRecord) {
	sort.SliceStable(requests, func(i, j int) bool {
		// non-resource < resource
		if requests[i].ResourceRequest != requests[j].ResourceRequest {
			return !requests[i].ResourceRequest
		}

		switch {
		case requests[i].ResourceRequest:
			// cluster-scoped < namespaced
			if n1, n2 := len(requests[i].Namespace) == 0, len(requests[j].Namespace) == 0; n1 != n2 {
				return n1
			}

			// unnamed < named
			if n1, n2 := len(requests[i].Name) == 0, len(requests[j].Name) == 0; n1 != n2 {
				return n1
			}

			// list < get
			if requests[i].Verb == "list" && requests[j].Verb == "get" {
				return true
			}
			if requests[i].Verb == "get" && requests[j].Verb == "list" {
				return false
			}

			// Sort by group,resource,subresource,namespace,name,verb
			if c := strings.Compare(requests[i].APIGroup, requests[j].APIGroup); c != 0 {
				return c < 0
			}
			if c := strings.Compare(requests[i].Resource, requests[j].Resource); c != 0 {
				return c < 0
			}
			if c := strings.Compare(requests[i].Subresource, requests[j].Subresource); c != 0 {
				return c < 0
			}
			if c := strings.Compare(requests[i].Namespace, requests[j].Namespace); c != 0 {
				return c < 0
			}
			if c := strings.Compare(requests[i].Name, requests[j].Name); c != 0 {
				return c < 0
			}
			if c := strings.Compare(requests[i].Verb, requests[j].Verb); c != 0 {
				return c < 0
			}

		case !requests[i].ResourceRequest:
			// Sort by verb,path
			if c := strings.Compare(requests[i].Verb, requests[j].Verb); c != 0 {
				return c < 0
			}
			if c := strings.Compare(requests[i].Path, requests[j].Path); c != 0 {
				return c < 0
			}
		}

		return false
	})
}

var (
	// Scheme knows about audit and rbac types
	Scheme = runtime.NewScheme()
	// Decoder knows how to decode audit and rbac objects
	Decoder runtime.Decoder
)

func init() {
	if err := rbacv1.AddToScheme(Scheme); err != nil {
		panic(err)
	}

	if err := auditv1.AddToScheme(Scheme); err != nil {
		panic(err)
	}

	if err := audit.AddToScheme(Scheme); err != nil {
		panic(err)
	}

	Decoder = serializer.NewCodecFactory(Scheme).UniversalDecoder()
}

// Output writes the specified object to the specified writer in "yaml" or "json" format
func Output(w io.Writer, obj runtime.Object, format string) error {
	var s *json.Serializer
	switch format {
	case "json":
		s = json.NewSerializerWithOptions(json.DefaultMetaFactory, Scheme, Scheme, json.SerializerOptions{false, true, false})
	case "yaml":
		s = json.NewSerializerWithOptions(json.DefaultMetaFactory, Scheme, Scheme, json.SerializerOptions{true, false, false})
	default:
		return fmt.Errorf("unknown format: %s", format)
	}

	codec := serializer.NewCodecFactory(Scheme).CodecForVersions(s, s, rbacv1.SchemeGroupVersion, rbacv1.SchemeGroupVersion)

	return codec.Encode(obj, w)
}

func EventToAttributes(event *audit.Event) authorizer.AttributesRecord {
	eventUser := &event.User
	if event.ImpersonatedUser != nil {
		eventUser = event.ImpersonatedUser
	}

	path := event.RequestURI
	if requestURL, err := url.ParseRequestURI(event.RequestURI); err == nil {
		path = requestURL.Path
	}

	attrs := authorizer.AttributesRecord{
		Verb: event.Verb,
		Path: path,
		User: &user.DefaultInfo{
			Name:   eventUser.Username,
			Groups: eventUser.Groups,
		},
	}

	if event.ObjectRef != nil {
		attrs.ResourceRequest = true
		attrs.Namespace = event.ObjectRef.Namespace
		attrs.Name = event.ObjectRef.Name
		attrs.Resource = event.ObjectRef.Resource
		attrs.Subresource = event.ObjectRef.Subresource
		attrs.APIGroup = event.ObjectRef.APIGroup
		attrs.APIVersion = event.ObjectRef.APIVersion
	}

	return attrs
}
