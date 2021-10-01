package format

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/alexeyco/simpletable"
	"github.com/ghodss/yaml"
	"github.com/spyzhov/ajson"
)

type Output struct {
	OutputMessage string
	JSONObject    interface{}
	OutputType    string
	TableColumns  []Column
	Filter        string

	root *ajson.Node
}

type Column struct {
	Name     string
	JSONPath string
	Expr     string
}

func (f *Output) Print() error {
	buf, err := json.Marshal(f.JSONObject)
	if err != nil {
		return err
	}

	f.root, err = ajson.Unmarshal(buf)
	if err != nil {
		return err
	}

	err = f.filterRows()
	if err != nil {
		return err
	}

	if f.OutputType == "table" {
		return f.outputTable()
	} else if f.OutputType == "json" {
		return f.outputJSON()
	} else if f.OutputType == "yaml" {
		return f.outputYAML()
	}

	return fmt.Errorf("unknown output type: %s", f.OutputType)
}

func (f *Output) Println() error {
	err := f.Print()
	if err != nil {
		return err
	}

	if f.OutputType == "table" {
		fmt.Println()
	}
	return nil
}

func (f *Output) filterRows() error {
	var filteredDocument []*ajson.Node

	if f.root.IsArray() {
		for _, ajsonRow := range f.root.MustArray() {
			meetsFilter, err := f.checkMeetsFilter(ajsonRow)
			if err != nil {
				return err
			}
			if !meetsFilter {
				continue
			}

			filteredDocument = append(filteredDocument, ajsonRow)
		}
	} else if f.root.IsObject() {
		meetsFilter, err := f.checkMeetsFilter(f.root)
		if err != nil {
			return err
		}
		if meetsFilter {
			filteredDocument = append(filteredDocument, f.root)
		}
	} else {
		return fmt.Errorf("json is not in the form of an object or array")
	}

	f.root = ajson.ArrayNode("", filteredDocument)
	return nil
}

func (f *Output) outputYAML() error {
	output := ajson.ObjectNode("", map[string]*ajson.Node{
		"msg": ajson.StringNode("", f.OutputMessage),
		"content": f.root,
	})

	document, err := ajson.Marshal(output)
	if err != nil {
		return err
	}

	yamlDocument, err := yaml.JSONToYAML(document)
	if err != nil {
		return err
	}

	fmt.Println("---")
	fmt.Println(string(yamlDocument))
	return nil
}

func (f *Output) outputJSON() error {
	output := ajson.ObjectNode("", map[string]*ajson.Node{
		"msg": ajson.StringNode("", f.OutputMessage),
		"content": f.root,
	})

	document, err := AJSONToIndentedJSON(output, " ", " ")
	if err != nil {
		return err
	}

	fmt.Println(string(document))
	return nil
}

func (f *Output) outputTable() error {
	table := simpletable.New()

	table.Header = &simpletable.Header{}
	for _, col := range f.TableColumns {
		table.Header.Cells = append(table.Header.Cells, &simpletable.Cell{
			Align: simpletable.AlignCenter,
			Text:  col.Name,
		})
	}

	table.Body = &simpletable.Body{}
	for _, ajsonRow := range f.root.MustArray() {
		row, err := f.formatPathRow(ajsonRow)
		if err != nil {
			return err
		}
		table.Body.Cells = append(table.Body.Cells, row)
	}

	table.SetStyle(simpletable.StyleCompactClassic)
	if f.OutputMessage != "" {
		fmt.Printf("[ %s ]\n", f.OutputMessage)
	}
	table.Println()

	return nil
}

func (f *Output) checkMeetsFilter(root *ajson.Node) (bool, error) {
	if f.Filter != "" {
		node, err := ajson.Eval(root, f.Filter)
		if err != nil {
			return false, err
		}
		return node.GetBool()
	}

	return true, nil
}

func (f *Output) formatPathRow(root *ajson.Node) ([]*simpletable.Cell, error) {
	formatPathRowError := func(err error) ([]*simpletable.Cell, error) {
		return []*simpletable.Cell{}, err
	}
	var err error

	var row []*simpletable.Cell
	for _, col := range f.TableColumns {
		var cell *simpletable.Cell

		if col.Expr != "" {
			cell, err = f.formatEvalExpr(root, col.JSONPath)
			if err != nil {
				return formatPathRowError(err)
			}
		} else if col.JSONPath != "" {
			cell, err = f.formatJSONPATH(root, col.JSONPath)
			if err != nil {
				return formatPathRowError(err)
			}
		} else {
			return formatPathRowError(fmt.Errorf("no expression or jsonpath set for column %s", col.Name))
		}

		row = append(row, cell)
	}

	return row, nil
}

func (f *Output) formatEvalExpr(root *ajson.Node, expr string) (*simpletable.Cell, error) {
	node, err := ajson.Eval(root, expr)
	if err != nil {
		return nil, err
	}

	value, err := node.Value()
	if err != nil {
		return nil, err
	}
	var column bytes.Buffer
	_, err = fmt.Fprint(&column, value)
	if err != nil {
		return nil, err
	}

	return &simpletable.Cell{Text: column.String()}, nil
}

func (f *Output) formatJSONPATH(root *ajson.Node, jsonPath string) (*simpletable.Cell, error) {
	buf, err := ajson.Marshal(root)
	if err != nil {
		return nil, err
	}
	col, err := ajson.JSONPath(buf, jsonPath)
	if err != nil {
		return nil, err
	}

	var column bytes.Buffer
	if len(col) == 1 {
		value, err := col[0].Value()
		if err != nil {
			return nil, err
		}
		_, err = fmt.Fprint(&column, value)
		if err != nil {
			return nil, err
		}
	} else {
		_, err = fmt.Fprint(&column, col)
		if err != nil {
			return nil, err
		}
	}

	return &simpletable.Cell{Text: column.String()}, nil
}

func AJSONToIndentedJSON(root *ajson.Node, prefix, indent string) ([]byte, error) {
	jsonBytes, err := ajson.Marshal(root)
	if err != nil {
		return nil, err
	}

	var jsonObj interface{}
	err = json.Unmarshal(jsonBytes, &jsonObj)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(jsonObj, prefix, indent)
}
