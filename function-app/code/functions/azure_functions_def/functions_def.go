package azure_functions_def

import (
	"fmt"

	"github.com/weka/go-cloud-lib/functions_def"
)

type AzureFuncDef struct {
	baseFunctionUrl string
	functionKey     string
}

func NewFuncDef(baseFunctionUrl, functionKey string) functions_def.FunctionDef {
	return &AzureFuncDef{
		baseFunctionUrl: baseFunctionUrl,
		functionKey:     functionKey,
	}
}

// each function takes json payload as an argument
// e.g. "{\"hostname\": \"$HOSTNAME\", \"type\": \"$message_type\", \"message\": \"$message\"}"
func (d *AzureFuncDef) GetFunctionCmdDefinition(name functions_def.FunctionName) string {
	functionUrl := d.baseFunctionUrl + string(name)
	var funcDef string
	if name == functions_def.Protect {
		funcDefTemplate := `
		function %s {
			echo "%s function is not implemented"
		}
		`
		funcDef = fmt.Sprintf(funcDefTemplate, name, name)
	} else if name == functions_def.JoinNfsFinalization {
		name = functions_def.JoinFinalization
		// edit json_data to add the missing "protocol":"nfs" field
		funcDefTemplate := `
		function %s {
			local json_data=$1
			json_data=$(echo $json_data | jq -c '.protocol="nfs"')

			curl --retry 10 %s?code=%s -H 'Content-Type:application/json' -d "$json_data"
		}
		`
		funcDef = fmt.Sprintf(funcDefTemplate, name, functionUrl, d.functionKey)
	} else {
		funcDefTemplate := `
		function %s {
			local json_data=$1
			curl --retry 10 %s?code=%s -H 'Content-Type:application/json' -d "$json_data"
		}
		`
		funcDef = fmt.Sprintf(funcDefTemplate, name, functionUrl, d.functionKey)
	}

	return funcDef
}
