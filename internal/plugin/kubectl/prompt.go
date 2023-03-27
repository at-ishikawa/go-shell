package kubectl

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var execCommand = exec.Command

func GetContext() (string, error) {
	cmd := execCommand("kubectl", "config", "current-context")
	if errors.Is(cmd.Err, exec.ErrNotFound) {
		return "", nil
	}

	kubeCtxResult, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		return "", fmt.Errorf("failed to get a current context from kubectx: %s %w", kubeCtxResult, err)
	}
	return strings.TrimSpace(string(kubeCtxResult)), nil
}

func GetNamespace(kubeCtx string) (string, error) {
	cmd := execCommand("kubectl", "config", "view", fmt.Sprintf("-o=jsonpath='{.contexts[?(@.name==\"%s\")].context.namespace}'", kubeCtx))
	if errors.Is(cmd.Err, exec.ErrNotFound) {
		return "", nil
	}

	kubeNamespaceResult, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get a current namespace from kubens: %s %w", kubeNamespaceResult, err)
	}
	return strings.Trim(string(kubeNamespaceResult), "'"), nil
}
