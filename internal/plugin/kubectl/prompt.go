package kubectl

import (
	"fmt"
	"os/exec"
	"strings"
)

func GetContext() (string, error) {
	kubeCtxResult, err := exec.Command("kubectl", "config", "current-context").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get a current context from kubectx: %s %w", kubeCtxResult, err)
	}
	return strings.TrimSpace(string(kubeCtxResult)), nil
}

func GetNamespace(kubeCtx string) (string, error) {
	kubeNamespaceResult, err := exec.Command("kubectl", "config", "view", fmt.Sprintf("-o=jsonpath='{.contexts[?(@.name==\"%s\")].context.namespace}'", kubeCtx)).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get a current namespace from kubens: %s %w", kubeNamespaceResult, err)
	}
	return strings.Trim(string(kubeNamespaceResult), "'"), nil
}
