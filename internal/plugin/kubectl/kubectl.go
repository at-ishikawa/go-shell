package kubectl

//go:generate go-shell-cli-option-parser kubectl -o "kubectloptions/kubectl_options.go"

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/at-ishikawa/go-shell/internal/completion"
	"github.com/at-ishikawa/go-shell/internal/plugin"
	"github.com/at-ishikawa/go-shell/internal/plugin/kubectl/kubectloptions"
	"github.com/ktr0731/go-fuzzyfinder"
)

const Cli = "kubectl"

var _ plugin.Plugin = new(KubeCtlPlugin)

type KubeCtlPlugin struct {
	completionUi *completion.Fzf
}

func NewKubeCtlPlugin(completionUi *completion.Fzf) plugin.Plugin {
	return &KubeCtlPlugin{
		completionUi: completionUi,
	}
}

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

func (k *KubeCtlPlugin) Command() string {
	return Cli
}

func (k *KubeCtlPlugin) Suggest(args []string) ([]string, error) {
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

	return k.searchByFzf(kubeCtlGetResult, namespace, resource, isMultipleResources)
}

func (k KubeCtlPlugin) searchByFzf(kubeCtlGetResult []byte,
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

	rows, err := k.completionUi.CompleteBytes(kubeCtlGetResult, fzfOptions)
	if err != nil {
		return []string{}, err
	}

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
