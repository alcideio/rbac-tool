package cmd

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MetadataFlag struct {
	metadata metav1.ObjectMeta
}

func (f *MetadataFlag) String() string {
	b, err := json.Marshal(f.metadata)
	if err != nil {
		return "failed to marshal metadata object"
	}
	return string(b)
}

func (f *MetadataFlag) Set(v string) error {
	f.metadata = metav1.ObjectMeta{}
	return json.Unmarshal([]byte(v), &f.metadata)
}

func (f *MetadataFlag) Type() string {
	return "json"
}
