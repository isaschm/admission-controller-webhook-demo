package admission

import (
	"context"
	"fmt"

	"golang.org/x/exp/slices"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	locationLabels = []string{"topology.gke.io/zone", "topology.kubernetes.io/region", "topology.kubernetes.io/zone"}
)

// Retrieves regions and zones of all nodes and returns locations as strings
// without differentiating between zone and region.
func GetNodeLocations(ctx context.Context) ([]string, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("could not create cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not create clientset: %w", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not retrieve nodes: %w", err)
	}

	var locations []string
	for _, node := range nodes.Items {
		for _, label := range locationLabels {
			if !slices.Contains(locations, node.Labels[label]) {
				locations = append(locations, node.Labels[label])
			}
		}
	}

	return locations, nil
}
