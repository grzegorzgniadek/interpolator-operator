//go:build e2e
// +build e2e

/*
Copyright 2026.

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

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/grzegorzgniadek/interpolator-operator/test/utils"
)

// namespace where the project is deployed in
const namespace = "interpolator-operator-system"

// serviceAccountName created for the project
const serviceAccountName = "interpolator-operator-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "interpolator-operator-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "interpolator-operator-metrics-binding"

// helmRelease is the Helm release name. It matches the chart name, so the chart's fullname
// resolves to "interpolator-operator" and resource names are identical to the kustomize deployment.
const helmRelease = "interpolator-operator"

// helmChartDir is the path to the Helm chart, relative to the project root.
const helmChartDir = "charts/interpolator-operator"

// deployMethod describes how the controller-manager is installed into and removed from the cluster.
// Every deploy method runs the same set of specs.
type deployMethod struct {
	name     string
	deploy   func()
	undeploy func()
}

var kustomizeDeploy = deployMethod{
	name: "kustomize",
	deploy: func() {
		By("installing CRDs")
		cmd := exec.Command("make", "install")
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", managerImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	},
	undeploy: func() {
		By("undeploying the controller-manager")
		cmd := exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)
	},
}

var helmDeploy = deployMethod{
	name: "helm",
	deploy: func() {
		repository, tag := splitImage(managerImage)

		By("installing the Helm chart")
		cmd := exec.Command(helmBinary(), "upgrade", "--install", helmRelease, helmChartDir,
			"--namespace", namespace,
			"--set", "manager.image.repository="+repository,
			"--set", "manager.image.tag="+tag,
			"--set", "manager.image.pullPolicy=IfNotPresent",
			"--wait",
			"--timeout", "5m",
		)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install the Helm chart")

		By("validating that the Helm release is deployed")
		cmd = exec.Command(helmBinary(), "status", helmRelease, "--namespace", namespace, "-o", "json")
		output, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to get Helm release status")
		var release struct {
			Info struct {
				Status string `json:"status"`
			} `json:"info"`
		}
		Expect(json.Unmarshal([]byte(output), &release)).To(Succeed())
		Expect(release.Info.Status).To(Equal("deployed"), "Helm release not deployed")
	},
	undeploy: func() {
		By("uninstalling the Helm release")
		cmd := exec.Command(helmBinary(), "uninstall", helmRelease, "--namespace", namespace, "--wait")
		_, _ = utils.Run(cmd)

		// The chart annotates CRDs with helm.sh/resource-policy=keep, so they survive the uninstall.
		By("removing CRDs kept by the Helm release")
		cmd = exec.Command("kubectl", "delete", "crd", "interpolators.interpolator.io", "--ignore-not-found")
		_, _ = utils.Run(cmd)
	},
}

var _ = describeManager(kustomizeDeploy)
var _ = describeManager(helmDeploy)

func describeManager(method deployMethod) bool {
	return Describe(fmt.Sprintf("Manager deployed with %s", method.name), Ordered, Label(method.name), func() {
		var controllerPodName string

		// Before running the tests, set up the environment by creating the namespace,
		// enforce the restricted security policy to the namespace, installing CRDs,
		// and deploying the controller.
		BeforeAll(func() {
			By("creating manager namespace")
			cmd := exec.Command("kubectl", "create", "ns", namespace)
			_, err := utils.Run(cmd)
			if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
				Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")
			}

			By("labeling the namespace to enforce the restricted security policy")
			cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
				"pod-security.kubernetes.io/enforce=restricted")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

			method.deploy()
		})

		// After all tests have been executed, clean up by removing the samples, undeploying the controller,
		// and deleting the namespace. Cluster-scoped leftovers are removed too, so the next deploy method
		// starts from a clean cluster.
		AfterAll(func() {
			By("cleaning up the curl pod for metrics")
			cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
			_, _ = utils.Run(cmd)

			By("cleaning up the metrics ClusterRoleBinding")
			cmd = exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName, "--ignore-not-found")
			_, _ = utils.Run(cmd)

			// Interpolators carry a finalizer, so they must be deleted while the controller is still running.
			By("removing example resources")
			cmd = exec.Command("kubectl", "delete", "-k", "config/samples", "--ignore-not-found", "--timeout=2m")
			_, _ = utils.Run(cmd)

			method.undeploy()

			By("removing manager namespace")
			cmd = exec.Command("kubectl", "delete", "ns", namespace, "--ignore-not-found")
			_, _ = utils.Run(cmd)
		})

		// After each test, check for failures and collect logs, events,
		// and pod descriptions for debugging.
		AfterEach(func() {
			specReport := CurrentSpecReport()
			if specReport.Failed() {
				By("Fetching controller manager pod logs")
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				controllerLogs, err := utils.Run(cmd)
				if err == nil {
					_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
				} else {
					_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
				}

				By("Fetching Kubernetes events")
				cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
				eventsOutput, err := utils.Run(cmd)
				if err == nil {
					_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
				} else {
					_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
				}

				By("Fetching curl-metrics logs")
				cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
				metricsOutput, err := utils.Run(cmd)
				if err == nil {
					_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
				} else {
					_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
				}

				By("Fetching controller manager pod description")
				cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
				podDescription, err := utils.Run(cmd)
				if err == nil {
					fmt.Println("Pod description:\n", podDescription)
				} else {
					fmt.Println("Failed to describe controller pod")
				}
			}
		})

		SetDefaultEventuallyTimeout(2 * time.Minute)
		SetDefaultEventuallyPollingInterval(time.Second)

		Context("Manager", func() {
			It("should run successfully", func() {
				By("validating that the controller-manager pod is running as expected")
				verifyControllerUp := func(g Gomega) {
					By("getting the name of the controller-manager pod")
					cmd := exec.Command("kubectl", "get",
						"pods", "-l", "control-plane=controller-manager",
						"-o", "go-template={{ range .items }}"+
							"{{ if not .metadata.deletionTimestamp }}"+
							"{{ .metadata.name }}"+
							"{{ \"\\n\" }}{{ end }}{{ end }}",
						"-n", namespace,
					)

					podOutput, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
					podNames := utils.GetNonEmptyLines(podOutput)
					g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
					controllerPodName = podNames[0]
					g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

					By("validating the pod's status")
					cmd = exec.Command("kubectl", "get",
						"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
						"-n", namespace,
					)
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
				}
				Eventually(verifyControllerUp).Should(Succeed())
			})

			It("should example resources be created successfully", func() {
				By("applying example resources")
				cmd := exec.Command("kubectl", "apply", "-k", "config/samples")
				_, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), "Failed to apply example resources")

				By("validating that the example resources are created")
				cmd = exec.Command("kubectl", "get", "interpolators", "-n", "default")
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), "Failed to get interpolator resources")

				By("validating that the example resources are synced")
				verifyInterpolatorsSynced := func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "interpolators.interpolator.io", "-n", "default",
						"-o", "jsonpath={.items[*].status.conditions[?(@.type=='Synced')].status}")
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())

					statuses := strings.Fields(strings.TrimSpace(output))
					g.Expect(statuses).To(HaveLen(2), "expected 2 Interpolator resources")
					g.Expect(statuses).To(HaveEach(Equal("True")), "Interpolator resources not synced: %v", statuses)

				}
				Eventually(verifyInterpolatorsSynced, 3*time.Minute, time.Second).Should(Succeed())

				expectedData := map[string]string{
					"username-rooted":  "ADMIN-with-permissions",
					"password-changed": "PASSWORD-with-something-added",
				}

				By("validating that the output ConfigMap contains interpolated data")
				verifyOutputConfigMap := func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "configmap", "testing-output", "-n", "default", "-o", "json")
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())

					var cm struct {
						Data map[string]string `json:"data"`
					}
					g.Expect(json.Unmarshal([]byte(output), &cm)).To(Succeed())
					g.Expect(cm.Data).To(Equal(expectedData))
				}
				Eventually(verifyOutputConfigMap, time.Minute, time.Second).Should(Succeed())

				By("validating that the output Secret contains interpolated data")
				verifyOutputSecret := func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "secret", "testing-output", "-n", "default", "-o", "json")
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())

					// []byte fields are base64-decoded by encoding/json
					var secret struct {
						Data map[string][]byte `json:"data"`
					}
					g.Expect(json.Unmarshal([]byte(output), &secret)).To(Succeed())

					decoded := make(map[string]string, len(secret.Data))
					for k, v := range secret.Data {
						decoded[k] = string(v)
					}
					g.Expect(decoded).To(Equal(expectedData))
				}
				Eventually(verifyOutputSecret, time.Minute, time.Second).Should(Succeed())
			})

			It("should ensure the metrics endpoint is serving metrics", func() {
				By("creating a ClusterRoleBinding for the service account to allow access to metrics")
				cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
					"--clusterrole=interpolator-operator-metrics-reader",
					fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
				)
				_, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

				By("validating that the metrics service is available")
				cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

				By("getting the service account token")
				token, err := serviceAccountToken()
				Expect(err).NotTo(HaveOccurred())
				Expect(token).NotTo(BeEmpty())

				By("ensuring the controller pod is ready")
				verifyControllerPodReady := func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "pod", controllerPodName, "-n", namespace,
						"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).To(Equal("True"), "Controller pod not ready")
				}
				Eventually(verifyControllerPodReady, 3*time.Minute, time.Second).Should(Succeed())

				By("verifying that the controller manager is serving the metrics server")
				verifyMetricsServerStarted := func(g Gomega) {
					cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).To(ContainSubstring("Serving metrics server"),
						"Metrics server not yet started")
				}
				Eventually(verifyMetricsServerStarted, 3*time.Minute, time.Second).Should(Succeed())

				// +kubebuilder:scaffold:e2e-metrics-webhooks-readiness

				By("creating the curl-metrics pod to access the metrics endpoint")
				cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
					"--namespace", namespace,
					"--image=curlimages/curl:latest",
					"--overrides",
					fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": [
								"for i in $(seq 1 30); do curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics && exit 0 || sleep 2; done; exit 1"
							],
							"securityContext": {
								"readOnlyRootFilesystem": true,
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccountName": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

				By("waiting for the curl-metrics pod to complete.")
				verifyCurlUp := func(g Gomega) {
					cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
						"-o", "jsonpath={.status.phase}",
						"-n", namespace)
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
				}
				Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

				By("getting the metrics by checking curl-metrics logs")
				verifyMetricsAvailable := func(g Gomega) {
					metricsOutput, err := getMetricsOutput()
					g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
					g.Expect(metricsOutput).NotTo(BeEmpty())
					g.Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
				}
				Eventually(verifyMetricsAvailable, 2*time.Minute).Should(Succeed())
			})

			// +kubebuilder:scaffold:e2e-webhooks-checks

			// TODO: Customize the e2e test suite with scenarios specific to your project.
			// Consider applying sample/CR(s) and check their status and/or verifying
			// the reconciliation by using the metrics, i.e.:
			// metricsOutput, err := getMetricsOutput()
			// Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
			// Expect(metricsOutput).To(ContainSubstring(
			//    fmt.Sprintf(`controller_runtime_reconcile_total{controller="%s",result="success"} 1`,
			//    strings.ToLower(<Kind>),
			// ))
		})
	})
}

// helmBinary returns the Helm binary to use, overridable with the HELM environment variable.
func helmBinary() string {
	if helm := os.Getenv("HELM"); helm != "" {
		return helm
	}
	return "helm"
}

// splitImage splits an image reference like "repo/name:tag" into repository and tag.
func splitImage(image string) (string, string) {
	i := strings.LastIndex(image, ":")
	if i <= strings.LastIndex(image, "/") {
		return image, "latest"
	}
	return image[:i], image[i+1:]
}

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	By("creating temporary file to store the token request")
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		By("executing kubectl command to create the token")
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		By("parsing the JSON output to extract the token")
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() (string, error) {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	return utils.Run(cmd)
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
