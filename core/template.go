package core

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig"
	"text/template"
	"github.com/pkg/errors"
)

// RenderTemplate renders the given template with params map
func RenderTemplate(tmpl string, params map[string]string) (string, error) {
	w := new(bytes.Buffer)
	funcs := template.FuncMap{
		"required": templateRequired,
	}

	tpl, err := template.New("test").
		Funcs(sprig.TxtFuncMap()).
		Funcs(funcs).
		Option("missingkey=zero").
		Parse(tmpl)
	if err != nil {
		return "", err
	}

	err = tpl.Execute(w, params)
	return w.String(), err
}

// RenderParams renders the given parameters recursively
func RenderParams(params map[string]string) (map[string]string, error) {
	renderedParams := make(map[string]string)

	for key, value := range params {
		renderedValue, err := renderParamValue(value, params)
		if err != nil {
			return nil, err
		}
		renderedParams[key] = renderedValue
	}

	return renderedParams, nil
}

func renderParamValue(value string, params map[string]string) (string, error) {
	return renderParamValueRecursively(value, params, 0)
}

func renderParamValueRecursively(value string, params map[string]string, recursionLevel int) (string,
	error) {
	const maxRecursionCount = 5

	if recursionLevel >= maxRecursionCount {
		return "", errors.Errorf("Cannot render template '%v'. Max recursion level (%v) exceeded.", value, maxRecursionCount)
	}

	renderedValue, err := RenderTemplate(value, params)
	if err != nil {
		return "", err
	}

	if renderedValue == value {
		return renderedValue, nil
	}

	return renderParamValueRecursively(renderedValue, params, recursionLevel+1)
}

func templateRequired(s string) (interface{}, error) {
	if len(s) > 0 {
		return s, nil
	}
	return s, fmt.Errorf("variable not provided to template")
}
