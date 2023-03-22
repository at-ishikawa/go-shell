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

var _ plugin.Plugin = (*KubeCtlPlugin)(nil)

type KubeCtlPlugin struct {
	completionUi completion.Completion
}

func NewKubeCtlPlugin(completionUi completion.Completion) plugin.Plugin {
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
		if i < len(args) {
			nextArg := args[i]
			if opt.HasDefaultValue {
				// todo: fix based on the default value type
				if nextArg == "true" || nextArg == "false" {
					continue
				}
			}
			resultOptions[opt.LongOption] = nextArg
			i = i + 1
		}

	}
	return result, resultOptions
}

func (k *KubeCtlPlugin) Command() string {
	return Cli
}

func (k *KubeCtlPlugin) GetContext(inputCommand string) (map[string]string, error) {
	result := map[string]string{}

	// TODO: replace the user defined context with the cluster
	context, err := GetContext()
	if err != nil {
		return result, err
	}
	result["context"] = context

	_, resultOptions := filterOptions(strings.Fields(inputCommand), kubectloptions.KubeCtlGlobalOptions)
	if namespace, ok := resultOptions["namespace"]; ok {
		result["namespace"] = namespace
	} else {
		namespace, err = GetNamespace(context)
		if err != nil {
			return result, err
		}
		result["namespace"] = namespace
	}
	return result, nil
}

func (k *KubeCtlPlugin) Suggest(arg plugin.SuggestArg) ([]string, error) {
	args := arg.Args
	if len(args) < 2 {
		return arg.Suggest(k.completionUi)
	}
	var namespace string
	var resultOptions map[string]string
	args, resultOptions = filterOptions(args, kubectloptions.KubeCtlGlobalOptions)
	if len(args) < 2 {
		return arg.Suggest(k.completionUi)
	}

	subCommand := args[1]
	if subCommand != "view-secret" {
		subCommandOptions, ok := kubectloptions.KubeCtlOptions[subCommand]
		if !ok {
			return arg.Suggest(k.completionUi)
		}
		if ns, ok := resultOptions["namespace"]; ok {
			namespace = ns
		}
		args, _ = filterOptions(args, subCommandOptions)
	}

	var resource string
	var isMultipleResources bool
	switch subCommand {
	case "view-secret":
		resource = "secrets"
		break
	case "rollout":
		if len(args) < 4 {
			return arg.Suggest(k.completionUi)
		}
		resource = args[3]
		break
	case "exec", "log", "logs":
		resource = "pods"
		break
	case "port-forward":
		resource = "pods,services"
		isMultipleResources = true
		break
	default:
		if len(args) < 3 {
			return arg.Suggest(k.completionUi)
		}
		resource = args[2]
		if resource == "events" {
			return arg.Suggest(k.completionUi)
		}
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

	return k.searchByCompletion(kubeCtlGetResult, namespace, resource, isMultipleResources)
}

func (k KubeCtlPlugin) searchByFzf(kubeCtlGetResult []byte,
	namespace string,
	resource string,
	isMultipleResources bool) ([]string, error) {
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
	fzfOptions := completion.FzfOption{
		Layout:         completion.FzfOptionLayoutReverse,
		PreviewWindow:  "down:50%",
		PreviewCommand: previewCommand,
	}
	if hasHeader {
		fzfOptions.HeaderLines = 1
	}

	/*
		rows, err := k.completionUi.CompleteMulti(strings.Split(string(kubeCtlGetResult), "\n"), fzfOptions)
		if err != nil {
			return []string{}, err
		}

		names := make([]string, len(rows))
		for i, row := range rows {
			columns := strings.Fields(row)
			names[i] = strings.TrimSpace(columns[0])
		}
		return names, nil
	*/
	return []string{}, nil
}

func (k KubeCtlPlugin) searchByCompletion(kubeCtlGetResult []byte,
	namespace string,
	resource string,
	isMultipleResources bool) ([]string, error) {
	var previewCommand string
	var header string
	kubectlResult := strings.Split(strings.TrimSpace(string(kubeCtlGetResult)), "\n")

	if !isMultipleResources {
		header = kubectlResult[0]
		kubectlResult = kubectlResult[1:]
	}
	if namespace != "" {
		previewCommand = previewCommand + " --namespace " + namespace
	}

	completeOptions := completion.CompleteOptions{
		Header: header,
		PreviewCommand: func(row int) (string, error) {
			name := strings.Fields(kubectlResult[row])[0]
			var previewCommandArgs []string
			if isMultipleResources {
				previewCommandArgs = []string{
					"describe",
					name,
				}
			} else {
				previewCommandArgs = []string{
					"describe",
					resource,
					name,
				}
			}

			output, err := exec.Command(Cli, previewCommandArgs...).Output()
			if err != nil {
				return "", err
			}
			if len(output) == 0 {
				return "", err
			}
			return string(output), err
		},
	}

	rows, err := k.completionUi.CompleteMulti(kubectlResult, completeOptions)
	if err != nil {
		return []string{}, err
	}
	if len(rows) == 0 {
		return []string{}, nil
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
