package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/inspektor-gadget/inspektor-gadget/cmd/kubectl-gadget/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	githubVersionOnce   sync.Once
	cachedGithubVersion string
	githubVersionError  error
)

// IsInspektorGadgetDeployed is a generic function to check if Inspektor Gadget is deployed in the cluster
// e.g. using kubectl-gadget, helm, or other means. It returns a boolean indicating if it is deployed,
// the namespace it is deployed in, and any error encountered
func IsInspektorGadgetDeployed(ctx context.Context) (bool, string, error) {
	restConfig, err := utils.KubernetesConfigFlags.ToRESTConfig()
	if err != nil {
		return false, "", fmt.Errorf("creating RESTConfig: %w", err)
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return false, "", fmt.Errorf("setting up trace client: %w", err)
	}

	opts := metav1.ListOptions{LabelSelector: "k8s-app=gadget"}
	pods, err := client.CoreV1().Pods("").List(ctx, opts)
	if err != nil {
		return false, "", fmt.Errorf("getting pods: %w", err)
	}
	if len(pods.Items) == 0 {
		log.Debug("No Inspektor Gadget pods found")
		return false, "", nil
	}

	var namespaces []string
	for _, pod := range pods.Items {
		if !slices.Contains(namespaces, pod.Namespace) {
			namespaces = append(namespaces, pod.Namespace)
		}
	}
	if len(namespaces) > 1 {
		log.Debug("Multiple namespaces found for Inspektor Gadget pods", "namespaces", namespaces)
		return false, "", fmt.Errorf("multiple namespaces found for Inspektor Gadget pods: %v", namespaces)
	}
	return true, namespaces[0], nil
}

// getChartVersion retrieves the version of the Inspektor Gadget Helm chart.
// It first attempts to get the version from GitHub releases, and if that fails,
// it falls back to the version from the build information.
func getChartVersion() string {
	if version, err := getLatestVersionFromGitHub(); err == nil {
		return version
	}
	return getChartVersionFromBuild()
}

// getChartVersionFromBuild retrieves the version of the Inspektor Gadget Helm chart from the build information.
func getChartVersionFromBuild() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/inspektor-gadget/inspektor-gadget" {
				if dep.Version != "" {
					return strings.TrimPrefix(dep.Version, "v")
				}
			}
		}
	}
	return "1.0.0-dev"
}

// getLatestVersionFromGitHub retrieves the version of the latest Inspektor Gadget release from GitHub.
func getLatestVersionFromGitHub() (string, error) {
	githubVersionOnce.Do(func() {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(defaultReleaseUrl)
		if err != nil {
			githubVersionError = fmt.Errorf("failed to get latest release: %w", err)
			return
		}
		defer func() {
			if err = resp.Body.Close(); err != nil {
				log.Error("closing response body", "error", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			githubVersionError = fmt.Errorf("failed to get latest release, status code: %d", resp.StatusCode)
			return
		}

		var release struct {
			TagName string `json:"tag_name"`
		}
		if err = json.NewDecoder(resp.Body).Decode(&release); err != nil {
			githubVersionError = fmt.Errorf("decoding latest release response: %w", err)
			return
		}
		cachedGithubVersion = strings.TrimPrefix(release.TagName, "v")
	})
	return cachedGithubVersion, githubVersionError
}
