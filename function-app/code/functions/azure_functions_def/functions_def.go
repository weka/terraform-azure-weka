package azure_functions_def

import (
	"fmt"
	"os"

	"github.com/weka/go-cloud-lib/functions_def"
)

type AzureFuncDef struct {
	baseFunctionUrl string
}

func NewFuncDef() functions_def.FunctionDef {
	baseFunctionUrl := fmt.Sprintf("%s:%s", os.Getenv("HTTP_SERVER_HOST"), os.Getenv("HTTP_SERVER_PORT"))
	return &AzureFuncDef{baseFunctionUrl: baseFunctionUrl}
}

// each function takes json payload as an argument
// e.g. "{\"hostname\": \"$HOSTNAME\", \"type\": \"$message_type\", \"message\": \"$message\"}"
func (d *AzureFuncDef) GetFunctionCmdDefinition(name functions_def.FunctionName) string {
	functionUrl := d.baseFunctionUrl + string(name)
	reportDefTemplate := `
	function %s {
		local json_data=$1
		curl %s -H 'Content-Type:application/json' -d "$json_data"
	}
	`
	reportDef := fmt.Sprintf(reportDefTemplate, name, functionUrl)
	return reportDef
}
