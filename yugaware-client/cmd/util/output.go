package util

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/alexeyco/simpletable"
	"github.com/ghodss/yaml"
	"github.com/spyzhov/ajson"
)

type OutputFormatter struct {
	OutputMessage string
	JSONObject    interface{}
	OutputType    string
	TableColumns  []Column
}

type OutputContainer struct {
	Message string      `json:"msg"`
	Content interface{} `json:"content"`
}

type Column struct {
	Name     string
	JSONPath string
}

func (f *OutputFormatter) Print() error {
	if f.OutputType == "table" {
		return f.OutputTable()
	} else if f.OutputType == "json" {
		return f.OutputJSON()
	} else if f.OutputType == "yaml" {
		return f.OutputYAML()
	}

	return fmt.Errorf("unknown output type: %s", f.OutputType)
}

func (f *OutputFormatter) OutputYAML() error {
	output := OutputContainer{
		Message: f.OutputMessage,
		Content: f.JSONObject,
	}
	bytes, err := yaml.Marshal(output)
	if err != nil {
		return err
	}

	fmt.Println(string(bytes))
	return nil
}

func (f *OutputFormatter) OutputJSON() error {
	output := OutputContainer{
		Message: f.OutputMessage,
		Content: f.JSONObject,
	}
	bytes, err := json.MarshalIndent(output, " ", " ")
	if err != nil {
		return err
	}

	fmt.Println(string(bytes))
	return nil
}

func (f *OutputFormatter) OutputTable() error {
	table := simpletable.New()

	table.Header = &simpletable.Header{}
	for _, col := range f.TableColumns {
		table.Header.Cells = append(table.Header.Cells, &simpletable.Cell{
			Align: simpletable.AlignCenter,
			Text:  col.Name,
		})
	}

	buf, err := json.Marshal(f.JSONObject)
	if err != nil {
		return err
	}
	//var unstructuredObject []map[string]interface{}
	root, err := ajson.Unmarshal(buf)
	if err != nil {
		return err
	}
	if root.IsArray() {
		table.Body = &simpletable.Body{}
		for _, ajsonRow := range root.MustArray() {
			row, err := f.formatJSONPathRow(ajsonRow)
			if err != nil {
				return err
			}
			table.Body.Cells = append(table.Body.Cells, row)
		}
	} else if root.IsObject() {
		row, err := f.formatJSONPathRow(root)
		if err != nil {
			return err
		}
		table.Body.Cells = append(table.Body.Cells, row)
	} else {
		return fmt.Errorf("json is not in the form of an object or array")
	}

	table.SetStyle(simpletable.StyleCompactClassic)
	if f.OutputMessage != "" {
		fmt.Printf("[ %s ]\n", f.OutputMessage)
	}
	table.Println()

	return nil
}

func (f *OutputFormatter) formatJSONPathRow(root *ajson.Node) ([]*simpletable.Cell, error) {
	var row []*simpletable.Cell
	for _, col := range f.TableColumns {
		buf, err := ajson.Marshal(root)
		if err != nil {
			return nil, err
		}
		col, err := ajson.JSONPath(buf, col.JSONPath)
		if err != nil {
			return nil, err
		}

		if len(col) == 1 {
			value, err := col[0].Value()
			if err != nil {
				return nil, err
			}
			var column bytes.Buffer
			_, err = fmt.Fprint(&column, value)
			if err != nil {
				return nil, err
			}
			row = append(row, &simpletable.Cell{
				Text: column.String(),
			})
		} else {
			var column bytes.Buffer
			_, err = fmt.Fprint(&column, col)
			if err != nil {
				return nil, err
			}
			row = append(row, &simpletable.Cell{
				Text: column.String(),
			})
		}
	}
	return row, nil
}
