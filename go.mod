module github.com/rjnienaber/prescript

go 1.22

require (
	github.com/creack/pty v1.1.24
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.7.0
	github.com/xeipuuv/gojsonschema v1.2.0
	go.uber.org/zap v1.16.0
	golang.org/x/term v0.18.0 // pinned: v0.45 needs go 1.25, and CI builds 1.22
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	go.uber.org/atomic v1.6.0 // indirect
	go.uber.org/multierr v1.5.0 // indirect
	golang.org/x/sys v0.18.0 // indirect; pinned with golang.org/x/term
)
