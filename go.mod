module github.com/qoryai/integration-template

go 1.27.1

require (
	github.com/qoryai/integrations v0.1.0
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
	golang.org/x/text v0.14.0
)

require (
	github.com/qoryai/runner v0.6.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Removed once qoryai/integrations v0.1.0 is tagged: until then the module is its
// checkout beside this one, ../../integrations/main.
replace github.com/qoryai/integrations => ../../integrations/main
