/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k8s

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	goyaml "github.com/go-yaml/yaml"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	kbuild "sigs.k8s.io/kustomize/kustomize/v4/commands/build"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// ApplyObjects parses a structured Kubernetes-compatible yaml file and applies
// its objects to a target Kubernetes cluster
func (kcl KubernetesClient) ApplyObjects(yamlData [][]byte) error {
	log.Info().Msgf("applying objects against kubernetes cluster")

	// RESTMapper to find GVR
	dc, err := discovery.NewDiscoveryClientForConfig(kcl.RestConfig)
	if err != nil {
		return fmt.Errorf("error creating discovery client: %w", err)
	}
	// Dynamic client
	dyn, err := dynamic.NewForConfig(kcl.RestConfig)
	if err != nil {
		return fmt.Errorf("error creating dynamic client: %w", err)
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	for _, resource := range yamlData {
		// Decode YAML manifest into unstructured.Unstructured
		obj := &unstructured.Unstructured{}
		_, gvk, err := decUnstructured.Decode(resource, nil, obj)
		if err != nil {
			return fmt.Errorf("error decoding data into unstructured object: %w", err)
		}

		// Find GVR
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("error finding GVR for %q: %w", gvk.Kind, err)
		}

		// REST interface for the GVR
		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			// namespaced resources should specify the namespace
			dr = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
		} else {
			// for cluster-wide resources
			dr = dyn.Resource(mapping.Resource)
		}

		// Marshal object into JSON
		data, err := json.Marshal(obj)
		if err != nil {
			return fmt.Errorf("error marshalling object %q into JSON: %w", obj.GetName(), err)
		}

		// Create or Update the object with server-side apply
		//
		//	types.ApplyPatchType indicates server-side apply
		//	FieldManager specifies the field owner ID
		_, err = dr.Patch(context.Background(), obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
			FieldManager: "kubefirst",
		})
		if err != nil {
			return fmt.Errorf("error applying %s/%s: %w", gvk.Kind, obj.GetName(), err)
		}

		log.Info().Msgf("applied %s %s", gvk.Kind, obj.GetName())
	}

	return nil
}

// KustomizeBuild parses a file path and returns manifests built via
// kustomization.yaml if present
//
// kustomizationDirectory should be a directory containing a kustomization.yaml
// file and subsequent configuration
//
// The return values is a string representation of parsed resources in yaml
func (kcl KubernetesClient) KustomizeBuild(kustomizationDirectory string) (*bytes.Buffer, error) {
	fSys := filesys.MakeFsOnDisk()

	buffer := new(bytes.Buffer)
	cmd := kbuild.NewCmdBuild(fSys, kbuild.MakeHelp("kubefirst", "internal kustomize build"), buffer)

	if err := cmd.RunE(cmd, []string{kustomizationDirectory}); err != nil {
		return nil, fmt.Errorf("error running kustomize build: %w", err)
	}

	return buffer, nil
}

// ReadYAMLFile reads a yaml file in the filesystem
func (kcl KubernetesClient) ReadYAMLFile(filepath string) (string, error) {
	dat, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	return string(dat), nil
}

// SplitYAMLFile takes a separated (---) yaml doc and returns [][]byte
func (kcl KubernetesClient) SplitYAMLFile(yamlData *bytes.Buffer) ([][]byte, error) {
	dec := goyaml.NewDecoder(bytes.NewReader(yamlData.Bytes()))

	var res [][]byte
	for {
		var value interface{}
		err := dec.Decode(&value)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("error decoding yaml: %w", err)
		}

		valueBytes, err := goyaml.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("error marshalling yaml: %w", err)
		}

		res = append(res, valueBytes)
	}

	return res, nil
}
