package kubectl

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func GetContext() (string, error) {
	kubeCtxResult, err := execCommand("kubectl", "config", "current-context")
	if errors.Is(err, exec.ErrNotFound) {
		return "", nil
	} else if err != nil {
		fmt.Println(err)
		return "", fmt.Errorf("failed to get a current context from kubectx: %s %w", kubeCtxResult, err)
	}

	return strings.TrimSpace(string(kubeCtxResult)), nil
}

func GetNamespace(kubeCtx string) (string, error) {
	kubeNamespaceResult, err := execCommand("kubectl", "config", "view", fmt.Sprintf("-o=jsonpath='{.contexts[?(@.name==\"%s\")].context.namespace}'", kubeCtx))
	if errors.Is(err, exec.ErrNotFound) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("failed to get a current namespace from kubens: %s %w", kubeNamespaceResult, err)
	}

	return strings.Trim(string(kubeNamespaceResult), "'"), nil
}
