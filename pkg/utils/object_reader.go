package utils

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
)

func ReadYamlManifest(r io.Reader) ([]runtime.Object, error) {
	decoded := []runtime.Object{}

	buf, err := ioutil.ReadAll(r)

	if err != nil {
		return nil, err
	}

	//klog.V(6).Infof("ReadAll '%v'", string(buf))

	bufSlice := bytes.Split(buf, []byte("\n---"))
	decoder := scheme.Codecs.UniversalDeserializer()

	for _, b := range bufSlice {
		obj, _, err := decoder.Decode(b, nil, nil)

		if err != nil {
			klog.V(6).Infof("failed to decode - %v - '%v'", err, string(b))
			continue
		}

		if obj == nil {
			klog.V(6).Infof("failed to decode - %v", string(b))
			continue
		}

		decoded = append(decoded, obj)
	}

	return decoded, nil
}

func ReadObjectList(r io.Reader) ([]runtime.Object, error) {
	buf, err := ioutil.ReadAll(r)

	if err != nil {
		return nil, err
	}

	decoder := scheme.Codecs.UniversalDeserializer()
	obj, gvk, err := decoder.Decode(buf, nil, nil)
	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, fmt.Errorf("Failed to decode")
	}

	switch o := obj.(type) {
	case *v1.List:
		if gvk.GroupKind().Kind != "List" {
			return nil, fmt.Errorf("Expected List Object Kind")
		}

		return convert(o.Items)
	default:
		return nil, fmt.Errorf("Failed to cast loaded object - '%v'", reflect.TypeOf(obj))
	}
}

func ReadObjectsFromFile(filename string) ([]runtime.Object, error) {
	objs := []runtime.Object{}

	var tee io.Reader
	var buf bytes.Buffer

	if filename != "-" {
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		tee = io.TeeReader(f, &buf)
	} else {
		tee = io.TeeReader(os.Stdin, &buf)
	}

	if l, err := ReadObjectList(tee); err == nil {
		klog.V(6).Infof("Loaded from Object List %v resources", len(l))
		objs = l
	} else {
		klog.V(6).Infof("Couldn't read Object List (%v) from %v ... trying to load as YAML", err, filename)
		if l, err := ReadYamlManifest(strings.NewReader(buf.String())); err == nil {
			klog.V(6).Infof("Loaded from YAML %v resources %v", filename, len(l))
			objs = l
		} else {
			return nil, fmt.Errorf("Failed to read kubernetes resources")
		}
	}

	return objs, nil
}

func convert(objs []runtime.RawExtension) ([]runtime.Object, error) {
	errs := []error{}
	decoded := []runtime.Object{}

	decoder := scheme.Codecs.UniversalDeserializer()

	for _, raw := range objs {
		obj, gvk, err := decoder.Decode(raw.Raw, nil, nil)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if obj == nil {
			errs = append(errs, fmt.Errorf("Object %+v decoded into nil", gvk))
			continue
		}

		decoded = append(decoded, obj)
	}

	return decoded, errors.NewAggregate(errs)
}
