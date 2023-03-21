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

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"golang.org/x/exp/slices"

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
)

// applyTransparencyLabeling implements the logic of our example admission controller webhook. For every pod that is created
// (outside of Kubernetes namespaces), it checks whether the necessary transparency tags are set in
// pod annotations. If not, it adds the tags with the value "unspecified"
func applyTransparencyLabeling(req *admission.AdmissionRequest) ([]patchOperation, error) {
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
	if _, _, err := universalDeserializer.Decode(raw, nil, &pod); err != nil {
		return nil, fmt.Errorf("could not deserialize pod object: %v", err)
	}

	// Create patch operations to add transparency information, if those labels are not set.
	var patches []patchOperation

	// Retrieve Annotations from Pod object
	annotations := pod.GetObjectMeta().GetAnnotations()
	if annotations == nil {
		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/metadata",
			Value: "{\"annotations\": {\"legalBasis\": \"unspecified\", \"legitimateInterest\": \"unspecified\", \"purposes\": \"unspecified\"}",
		})
	} else if annotations != nil {

		// Check if Transparency tags are present in the annotation, if not add them
		for _, label := range transparencyTags {
			if _, ok := annotations[label]; !ok {
				annotations[label] = "unspecified"
				log.Printf("Transparency tag %v added to annotations", label)
			}
		}

		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/metadata/annotations",
			Value: annotations,
		})
	}

	return patches, nil
}

func getNodeLocations() []string {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Could not create cluster config: %v", err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Could not create clientset: %v", err.Error())
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Could not retrieve nodes: %v", err.Error())
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
	return locations
}

func main() {
	certPath := filepath.Join(tlsDir, tlsCertFile)
	keyPath := filepath.Join(tlsDir, tlsKeyFile)

	getNodeLocations()

	mux := http.NewServeMux()
	mux.Handle("/mutate", admitFuncHandler(applyTransparencyLabeling))
	mux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server := &http.Server{
		// We listen on port 8443 such that we do not need root privileges or extra capabilities for this server.
		// The Service object will take care of mapping this port to the HTTPS port 443.
		Addr:    ":8443",
		Handler: mux,
	}
	log.Fatal(server.ListenAndServeTLS(certPath, keyPath))
}
