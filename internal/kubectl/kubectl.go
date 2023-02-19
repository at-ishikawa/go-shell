package kubectl

//go:generate go-shell-cli-option-parser kubectl -o "kubectloptions/kubectl_options.go"

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/at-ishikawa/go-shell/internal/kubectl/kubectloptions"
	"github.com/ktr0731/go-fuzzyfinder"
)

const kubeCtlCli = "kubectl"

func filterOptions(args []string, cliOptions []kubectloptions.CLIOption) ([]string, map[string]string) {
	result := make([]string, 0)
	resultOptions := make(map[string]string)

	optionMap := make(map[string]kubectloptions.CLIOption)
	for _, opt := range cliOptions {
		optionMap["-"+opt.ShortOption] = opt
		optionMap["--"+opt.LongOption] = opt
	}

	i := 0
	for i < len(args) {
		arg := args[i]
		i = i + 1
		if !strings.HasPrefix(arg, "-") {
			result = append(result, arg)
			continue
		}
		opt, ok := optionMap[arg]
		if !ok {
			continue
		}

		resultOptions[opt.LongOption] = ""

		if opt.HasDefaultValue && i < len(args) {
			nextArg := args[i]
			resultOptions[opt.LongOption] = nextArg
			i = i + 1
		}
	}
	return result, resultOptions
}

func Suggest(args []string) ([]string, error) {
	if len(args) < 2 {
		return []string{}, nil
	}
	var namespace string
	var resultOptions map[string]string
	args, resultOptions = filterOptions(args, kubectloptions.KubeCtlGlobalOptions)

	subCommand := args[1]
	subCommandOptions, ok := kubectloptions.KubeCtlOptions[subCommand]
	if !ok {
		// unsupported commands
		return []string{}, nil
	}
	if ns, ok := resultOptions["namespace"]; ok {
		namespace = ns
	}

	args, _ = filterOptions(args, subCommandOptions)
	var resource string
	switch subCommand {
	case "exec":
	case "log", "logs":
		resource = "pods"
		break
	case "port-forward":
		resource = "pods,services"
		break
	default:
		if len(args) < 3 {
			return []string{}, nil
		}
		resource = args[2]
	}

	suggestOptions := []string{
		"get",
		resource,
	}
	if namespace != "" {
		suggestOptions = append(suggestOptions, "-n", namespace)
	}
	result, err := exec.Command(kubeCtlCli, suggestOptions...).CombinedOutput()
	if err != nil {
		fmt.Println(string(result))
		return []string{}, err
	}
	lines := strings.Split(string(result), "\n")
	index, err := fuzzyfinder.Find(
		lines,
		func(i int) string {
			return lines[i]
		},
		fuzzyfinder.WithHeader(strings.Join(suggestOptions, " ")),
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			return lines[i]
		}))

	return []string{
		strings.Split(lines[index], " ")[0],
	}, nil
}
