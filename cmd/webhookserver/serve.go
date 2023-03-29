/*
Copyright (c) 2019 StackRox Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhookserver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"golang.org/x/exp/slices"

	admissionController "github.com/isaschm/admission-controller-webhook-demo/internal/admission"
	"github.com/isaschm/admission-controller-webhook-demo/internal/transparency"
	admission "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	tlsDir      = `/run/secrets/tls`
	tlsCertFile = `tls.crt`
	tlsKeyFile  = `tls.key`
)

var (
	podResource = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}
)

var (
	transparencyTags = []string{"purposes", "legitimateInterest", "legalBasis"}

	addEmptyAnnotationsOp = admissionController.PatchOperation{
		Op:    "add",
		Path:  "/metadata",
		Value: "\"annotations\": {\"legalBasis\": \"unspecified\", \"legitimateInterest\": \"unspecified\", \"purposes\": \"unspecified\"}",
	}
)

func applyTransparencyLabelerForLocations(locations []string) admissionController.AdmitFunc {
	// applyTransparencyLabeling implements the logic of our example admission controller webhook. For every pod that is created
	// (outside of Kubernetes namespaces), it checks whether the necessary transparency tags are set in
	// pod annotations. If not, it adds the tags with the value "unspecified"
	return func(req *admission.AdmissionRequest) ([]admissionController.PatchOperation, error) {
		// This handler should only get called on Pod objects as per the MutatingWebhookConfiguration in the YAML file.
		// However, if (for whatever reason) this gets invoked on an object of a different kind, issue a log message but
		// let the object request pass through otherwise.
		if req.Resource != podResource {
			log.Printf("expect resource to be %s", podResource)
			return nil, nil
		}

		// Parse the Pod object.
		raw := req.Object.Raw
		pod := corev1.Pod{}
		if _, _, err := admissionController.UniversalDeserializer.Decode(raw, nil, &pod); err != nil {
			return nil, fmt.Errorf("could not deserialize pod object: %v", err)
		}

		// Retrieve Labels from Pod object
		labels := pod.GetLabels()
		if labels["deployOutsideOfEU"] == "false" {
			if slices.ContainsFunc(locations, func(s string) bool {
				return strings.HasPrefix(s, "europe-west1")
			}) {
				return nil, errors.New("resource cannot be deployed outside of EU")
			}
		}

		// Create patch operations to add transparency information, if those labels are not set.
		var patches []admissionController.PatchOperation

		// Retrieve Annotations from Pod object
		annotations := pod.GetObjectMeta().GetAnnotations()

		if annotations == nil {
			patches = append(patches, addEmptyAnnotationsOp)
			annotations = make(map[string]string)
		}

		tags, err := transparency.DecodeTags(annotations)
		if err != nil {
			return patches, fmt.Errorf("get tags from annotations: %w", err)
		}

		encodedTags, err := tags.Encode()
		if err != nil {
			return nil, err
		}

		// The last operation is processed first, which means we need to prepend
		// operations that depend on adding the annotations tag
		patches = append([]admissionController.PatchOperation{admissionController.PatchOperation{
			Op:    "add",
			Path:  "/metadata/annotations",
			Value: encodedTags,
		}}, patches...)

		return patches, nil
	}
}

// Retrieves regions and zones of all nodes and returns locations as strings
// without differentiating between zone and region.
func getNodeLocations() ([]string, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("could not create cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not create clientset: %w", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not retrieve nodes: %w", err)
	}

	locationLabels := []string{"topology.gke.io/zone", "topology.kubernetes.io/region", "topology.kubernetes.io/zone"}
	var locations []string
	for _, node := range nodes.Items {
		for _, label := range locationLabels {
			if !slices.Contains(locations, node.Labels[label]) {
				locations = append(locations, node.Labels[label])
			}
		}
	}

	log.Printf("%v\n", locations)
	return locations, nil
}

func ExecuteServe() error {
	certPath := filepath.Join(tlsDir, tlsCertFile)
	keyPath := filepath.Join(tlsDir, tlsKeyFile)

	locations, err := getNodeLocations()
	if err != nil {
		return fmt.Errorf("fetch node locations: %w", err)
	}

	admitHandler := applyTransparencyLabelerForLocations(locations)

	mux := http.NewServeMux()
	mux.Handle("/mutate", admissionController.AdmitFuncHandler(admitHandler))
	mux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server := &http.Server{
		// We listen on port 8443 such that we do not need root privileges or extra capabilities for this server.
		// The Service object will take care of mapping this port to the HTTPS port 443.
		Addr:    ":8443",
		Handler: withLogging(log.Default())(mux),
	}

	if err := server.ListenAndServeTLS(certPath, keyPath); err != nil {
		return err
	}

	return nil
}

func withLogging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Println(r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			next.ServeHTTP(w, r)
		})
	}
}
