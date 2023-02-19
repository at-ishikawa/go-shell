
package kubectloptions

var KubeCtlGetCommandOptions = []CLIOption{

	{
		ShortOption: "A",
		LongOption: "all-namespaces",
		HasDefaultValue: true,
	},

	{
		ShortOption: "",
		LongOption: "allow-missing-template-keys",
		HasDefaultValue: true,
	},

	{
		ShortOption: "",
		LongOption: "chunk-size",
		HasDefaultValue: false,
	},

	{
		ShortOption: "",
		LongOption: "field-selector",
		HasDefaultValue: false,
	},

	{
		ShortOption: "f",
		LongOption: "filename",
		HasDefaultValue: false,
	},

	{
		ShortOption: "",
		LongOption: "ignore-not-found",
		HasDefaultValue: true,
	},

	{
		ShortOption: "k",
		LongOption: "kustomize",
		HasDefaultValue: false,
	},

	{
		ShortOption: "L",
		LongOption: "label-columns",
		HasDefaultValue: false,
	},

	{
		ShortOption: "",
		LongOption: "no-headers",
		HasDefaultValue: true,
	},

	{
		ShortOption: "o",
		LongOption: "output",
		HasDefaultValue: false,
	},

	{
		ShortOption: "",
		LongOption: "output-watch-events",
		HasDefaultValue: true,
	},

	{
		ShortOption: "",
		LongOption: "raw",
		HasDefaultValue: false,
	},

	{
		ShortOption: "R",
		LongOption: "recursive",
		HasDefaultValue: true,
	},

	{
		ShortOption: "l",
		LongOption: "selector",
		HasDefaultValue: false,
	},

	{
		ShortOption: "",
		LongOption: "server-print",
		HasDefaultValue: true,
	},

	{
		ShortOption: "",
		LongOption: "show-kind",
		HasDefaultValue: true,
	},

	{
		ShortOption: "",
		LongOption: "show-labels",
		HasDefaultValue: true,
	},

	{
		ShortOption: "",
		LongOption: "show-managed-fields",
		HasDefaultValue: true,
	},

	{
		ShortOption: "",
		LongOption: "sort-by",
		HasDefaultValue: false,
	},

	{
		ShortOption: "",
		LongOption: "subresource",
		HasDefaultValue: false,
	},

	{
		ShortOption: "",
		LongOption: "template",
		HasDefaultValue: false,
	},

	{
		ShortOption: "w",
		LongOption: "watch",
		HasDefaultValue: true,
	},

	{
		ShortOption: "",
		LongOption: "watch-only",
		HasDefaultValue: true,
	},

}
