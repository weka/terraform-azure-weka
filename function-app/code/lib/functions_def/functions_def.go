package functions_def

import (
	"fmt"
)

// each function takes json payload as an argument
// e.g. "{\"hostname\": \"$HOSTNAME\", \"type\": \"$message_type\", \"message\": \"$message\"}"
type FunctionName string

const (
	Clusterize              FunctionName = "clusterize"
	ClusterizeFinalizaition FunctionName = "clusterize_finalization"
	Deploy                  FunctionName = "deploy"
	Protect                 FunctionName = "protect"
	Report                  FunctionName = "report"
	Join                    FunctionName = "join"
	JoinFinalization        FunctionName = "join_finalization"
)

type FunctionDef interface {
	GetFunctionCmdDefinition(name FunctionName) string
}

type AzureFuncDef struct {
	baseFunctionUrl string
	functionKey     string
}

func NewFuncDef(baseFunctionUrl, functionKey string) FunctionDef {
	return &AzureFuncDef{
		baseFunctionUrl: baseFunctionUrl,
		functionKey:     functionKey,
	}
}

func (d *AzureFuncDef) GetFunctionCmdDefinition(name FunctionName) string {
	functionUrl := d.baseFunctionUrl + string(name)
	reportDefTemplate := `
	function %s {
		local json_data=$1
		curl %s?code=%s -H 'Content-Type:application/json' -d "$json_data"
	}
	`
	reportDef := fmt.Sprintf(reportDefTemplate, name, functionUrl, d.functionKey)
	return reportDef
}
