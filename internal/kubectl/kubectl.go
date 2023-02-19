package kubectl

//go:generate go-shell-cli-option-parser kubectl -o "kubectloptions/kubectl_options.go"

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"

	"github.com/at-ishikawa/go-shell/internal/kubectl/kubectloptions"
)

const Cli = "kubectl"

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
	var isMultipleResources bool
	switch subCommand {
	case "exec":
	case "log", "logs":
		resource = "pods"
		break
	case "port-forward":
		resource = "pods,services"
		isMultipleResources = true
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
	kubeCtlGetResult, err := exec.Command(Cli, suggestOptions...).CombinedOutput()
	if err != nil {
		fmt.Println(string(kubeCtlGetResult))
		return []string{}, err
	}

	return searchByFzf(string(kubeCtlGetResult), namespace, resource, isMultipleResources)
}

func searchByFzf(kubeCtlGetResult string,
	namespace string,
	resource string,
	isMultipleResources bool) ([]string, error) {
	fzfOptions := []string{
		"--inline-info",
		"--multi",
		"--layout reverse",
		"--preview-window down:70%",
		"--bind ctrl-k:kill-line,ctrl-alt-t:toggle-preview,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down",
	}
	var previewCommand string
	var hasHeader bool
	if isMultipleResources {
		previewCommand = fmt.Sprintf("%s describe {1}", Cli)
		hasHeader = false
	} else {
		hasHeader = true
		previewCommand = fmt.Sprintf("%s describe %s {1}", Cli, resource)
	}
	if namespace != "" {
		previewCommand = previewCommand + " --namespace " + namespace
	}
	fzfOptions = append(fzfOptions,
		fmt.Sprintf("--preview '%s'", previewCommand),
	)
	if hasHeader {
		fzfOptions = append(fzfOptions, "--header-lines 1")
	}

	command := fmt.Sprintf("echo '%s' | fzf %s", kubeCtlGetResult, strings.Join(fzfOptions, " "))
	execCmd := exec.Command("sh", "-c", command)
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin
	out, err := execCmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Script canceled by Ctrl-c
			// Only for bash?: http://tldp.org/LDP/abs/html/exitcodes.html
			if exitErr.ExitCode() == 130 {
				return []string{}, nil
			}
		}
		return []string{}, fmt.Errorf("failed to run the command %s: %w", command, err)
	}

	rows := strings.Split(strings.TrimSpace(string(out)), "\n")
	names := make([]string, len(rows))
	for i, row := range rows {
		columns := strings.Fields(row)
		names[i] = strings.TrimSpace(columns[0])
	}

	return names, nil
}

func searchByFzfFinder(result string) ([]string, error) {
	lines := strings.Split(result, "\n")
	index, err := fuzzyfinder.Find(
		lines,
		func(i int) string {
			return lines[i]
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			return lines[i]
		}))

	return []string{
		strings.Split(lines[index], " ")[0],
	}, err
}
