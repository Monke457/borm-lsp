package analysis

import (
	"encoding/csv"
	"fmt"
	"os"
)

type BormFunction struct {
	Group string
	Namespace string
	Name string
	Definition string
	Description string	
}

func NewBormFunction(group, ns, retval, name, params, desc string) BormFunction {
	definition := fmt.Sprintf("%s %s(%s) {}", retval, name, params)
	return BormFunction{
		Group: group,
		Namespace: ns,
		Name: name,
		Definition: definition,
		Description: desc,
	}
}

func FindFunctionByName(functions []BormFunction, name string) (*BormFunction, bool) {
	for _, function := range functions {
		if function.Name == name {
			return &function, true 
		}
	}
	return nil, false
}

func FindFunctionsByGroup(functions []BormFunction, group string) []BormFunction {
	groupFuncs := []BormFunction{}
	groupFound := false
	for _, function := range functions {
		if function.Group == group {
			groupFound = true
			groupFuncs = append(groupFuncs, function)
		}
		if groupFound {
			break
		}
	}
	return groupFuncs
}

func FindFunctionsByNamespace(functions []BormFunction, ns string) []BormFunction {
	nsFuncs := []BormFunction{}
	nsFound := false
	for _, function := range functions {
		if function.Namespace == ns {
			nsFound = true
			nsFuncs = append(nsFuncs, function)
		}
		if nsFound {
			break
		}
	}
	return nsFuncs
}

func ReadFunctionsFromFile(filename string) ([]BormFunction, error) {
	functions := []BormFunction{}
	file, err := os.Open(filename)
	if err != nil {
		return functions, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return functions, err
	}

	var group, ns string
	for _, record := range records {
		if record[0] != "" {
			group = record[0]
		}
		if record[1] != "" {
			ns = record[1]
		}
		retval, name, params, desc := record[2], record[3], record[4], record[5]
		functions = append(functions, NewBormFunction(group, ns, retval, name, params, desc))
	}

	return functions, nil
}
