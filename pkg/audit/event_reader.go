package audit

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apiserver/pkg/apis/audit"
	auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
	"k8s.io/klog"
	rbacv1helper "k8s.io/kubernetes/pkg/apis/rbac/v1"
)

type StreamObject struct {
	Obj runtime.Object
	Err error
}

func ReadAuditEvents(sources []string, filters ...func(*audit.Event) bool) (<-chan *StreamObject, error) {
	streams, errs := openStreams(sources)

	klog.V(7).Infof("opening '%v' source(s) - %v", len(streams), errs)

	results := readStreams(streams)
	results = flattenEventLists(results)
	results = normalizeEventType(results, Scheme, Scheme)
	results = filterEvents(results, filters...)

	return results, errors.NewAggregate(errs)
}

func openStreams(sources []string) ([]io.ReadCloser, []error) {
	streams := []io.ReadCloser{}
	errors := []error{}

	klog.V(5).Infof("opening readStreams(s) - %v", sources)

	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	for _, source := range sources {
		if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
			req, err := http.NewRequest("GET", source, nil)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			req.Header.Set("User-Agent", "alcideio/rbac-tool "+goruntime.GOOS+"/"+goruntime.GOARCH)

			resp, err := client.Do(req)
			if err != nil {
				errors = append(errors, err)
			} else if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				errors = append(errors, fmt.Errorf("error fetching %s: %d", source, resp.StatusCode))
			} else {
				streams = append(streams, resp.Body)
			}

			continue
		}

		if source == "-" {
			streams = append(streams, os.Stdin)
			continue
		}

		fstat, err := os.Stat(source)

		if err != nil {
			klog.V(5).Infof("failed to stat file %v - %v", source, err)
			errors = append(errors, err)
			continue
		}

		if isDir := fstat.IsDir(); !isDir {
			f, err := os.Open(source)
			if err != nil {
				errors = append(errors, err)
			} else {
				streams = append(streams, f)
			}
		}

		err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				klog.V(5).Infof("skip directory object %v", path)
				return nil
			}

			f, err := os.Open(path)
			if err != nil {
				errors = append(errors, err)
			} else {
				streams = append(streams, f)
			}
			return nil
		})

		if err != nil {
			klog.V(5).Infof("failed to load from dir %v - %v", source, err)
			errors = append(errors, err)
			continue
		}

	}

	return streams, errors
}

// decoder can decode streaming json, yaml docs, single json objects, single yaml objects
type decoder interface {
	Decode(into interface{}) error
}

func streamingDecoder(r io.ReadCloser) decoder {
	buffer := bufio.NewReaderSize(r, 4096)
	b, _ := buffer.Peek(1)
	if string(b) == "{" || string(b) == "[" {
		return json.NewDecoder(buffer)
	} else {
		return yaml.NewYAMLToJSONDecoder(buffer)
	}
}

func readStreams(sources []io.ReadCloser) <-chan *StreamObject {
	out := make(chan *StreamObject)

	wg := &sync.WaitGroup{}
	for i := range sources {
		wg.Add(1)
		go func(r io.ReadCloser) {
			//klog.V(5).Infof("opening readStreams %v", r)
			defer wg.Done()
			defer r.Close()
			d := streamingDecoder(r)
			for {
				obj := &unstructured.Unstructured{}
				err := d.Decode(obj)

				switch {
				case err == io.EOF:
					return
				case err != nil:
					klog.V(5).Infof("Fail %v", err)
					out <- &StreamObject{Err: err}
				default:
					//klog.V(7).Infof("Add %v", pretty.Sprint(obj))
					out <- &StreamObject{Obj: obj}
				}
			}
		}(sources[i])
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

func flattenEventLists(in <-chan *StreamObject) <-chan *StreamObject {
	out := make(chan *StreamObject)

	v1List := metav1.SchemeGroupVersion.WithKind("List")
	v1EventList := auditv1.SchemeGroupVersion.WithKind("EventList")

	go func() {
		defer close(out)
		for result := range in {
			if result.Err != nil {
				out <- result
				continue
			}

			objGvk := result.Obj.GetObjectKind().GroupVersionKind()

			switch objGvk {
			case v1List, v1EventList:
				data, err := json.Marshal(result.Obj)
				if err != nil {
					out <- &StreamObject{Err: err}
					continue
				}

				list := &unstructured.UnstructuredList{}
				if err := list.UnmarshalJSON(data); err != nil {
					out <- &StreamObject{Err: err}
					continue
				}

				for i, _ := range list.Items {
					//klog.V(7).Infof("Add - %v", pretty.Sprint(list.Items[i]))
					out <- &StreamObject{Obj: &list.Items[i]}
				}
			default:
				//klog.V(7).Infof("Not a list - %v", result.Obj.GetObjectKind().GroupVersionKind())
				out <- result
				continue
			}
		}
	}()
	return out
}

func normalizeEventType(in <-chan *StreamObject, creator runtime.ObjectCreater, convertor runtime.ObjectConvertor) <-chan *StreamObject {
	out := make(chan *StreamObject)

	go func() {
		defer close(out)
		for result := range in {
			if result.Err != nil {
				out <- result
				continue
			}

			//klog.V(7).Infof("[BEFORE] normalizeEventType %v", pretty.Sprint(result.Obj))
			typed, err := creator.New(result.Obj.GetObjectKind().GroupVersionKind())
			if err != nil {
				out <- &StreamObject{Err: err}
				continue
			}

			unstructuredObject, ok := result.Obj.(*unstructured.Unstructured)
			if !ok {
				out <- &StreamObject{Err: fmt.Errorf("expected *unstructured.Unstructured, got %T", result.Obj)}
			}

			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, typed); err != nil {
				out <- &StreamObject{Err: err}
				continue
			}

			objGvk := typed.GetObjectKind().GroupVersionKind()

			gv := objGvk.GroupVersion()
			if gv.Version == "" || gv.Version == runtime.APIVersionInternal {
				out <- &StreamObject{Obj: typed}
				continue
			}

			gv.Version = runtime.APIVersionInternal
			converted, err := convertor.ConvertToVersion(typed, gv)
			if err != nil {
				out <- &StreamObject{Err: err}
				continue
			}

			//event := converted.(*audit.Event)
			//klog.V(7).Infof("[%v] event converetd by filter [eventId=%v]", event.User.Username, event.AuditID)
			out <- &StreamObject{Obj: converted}
		}
	}()
	return out
}

func filterEvents(in <-chan *StreamObject, filters ...func(*audit.Event) bool) <-chan *StreamObject {
	out := make(chan *StreamObject)

	go func() {
		defer close(out)
		for result := range in {
			if result.Err != nil {
				out <- result
				continue
			}

			event, ok := result.Obj.(*audit.Event)
			if !ok {
				out <- &StreamObject{Err: fmt.Errorf("expected *audit.Event, got %T", result.Obj)}
				continue
			}

			include := true
			for _, filter := range filters {
				include = filter(event)
				if !include {
					break
				}
			}

			if include {
				out <- result
			} else {
				klog.V(7).Infof("[%v] event dropped by filter [eventId=%v]", event.User.Username, event.AuditID)
			}
		}
	}()

	return out
}

func GetDiscoveryRoles() RBACObjects {
	return RBACObjects{
		ClusterRoles: []*rbacv1.ClusterRole{
			&rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "system:discovery"},
				Rules: []rbacv1.PolicyRule{
					rbacv1helper.NewRule("get").URLs("/healthz", "/version", "/swagger*", "/openapi*", "/api*").RuleOrDie(),
				},
			},
		},
		ClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
			&rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{Name: "system:discovery"},
				Subjects: []rbacv1.Subject{
					{Kind: rbacv1.GroupKind, APIGroup: rbacv1.GroupName, Name: "system:authenticated"},
					{Kind: rbacv1.GroupKind, APIGroup: rbacv1.GroupName, Name: "system:unauthenticated"},
				},
				RoleRef: rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: "system:discovery"},
			},
		},
	}
}
