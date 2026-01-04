package output

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/gookit/color"
)

func PrintColoredHeaders(headers map[string][]string) {
	for key, values := range headers {
		for _, value := range values {
			color.Bold.Printf("%s: ", key)
			color.Cyan.Printf("%s\n", value)
		}
	}
}

func PrintColoredBody(body []byte) {
	var data interface{}
	err := json.Unmarshal(body, &data)
	if err == nil {
		printColoredJSON(data, 0)
		fmt.Println()
		return
	}

	// If parsing fails, print the raw body as plain text
	fmt.Println(string(body))
}

func printColoredJSON(data interface{}, indent int) {
	keyColor := color.New(color.FgLightGreen)
	valueColor := color.New(color.FgLightCyan)
	indentStr := ""
	for i := 0; i < indent; i++ {
		indentStr += "  "
	}

	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			fmt.Print(indentStr)
			keyColor.Printf("%s: ", key)
			if m, ok := value.(map[string]interface{}); ok {
				fmt.Println()
				printColoredJSON(m, indent+1)
			} else if s, ok := value.([]interface{}); ok {
				fmt.Println()
				printColoredJSON(s, indent+1)
			} else {
				valueColor.Println(value)
			}
		}
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				printColoredJSON(m, indent)
				fmt.Println()
			} else {
				fmt.Print(indentStr)
				valueColor.Println(item)
			}
		}
	default:
		valueColor.Println(v)
	}
}

func WriteToFile(fileName string, data []byte) error {
	if fileName == "" {
		return errors.New("missing file name")
	}

	// Use O_TRUNC instead of O_APPEND for typical -o behavior,
	// or keep O_APPEND if that's what's intended.
	// Usually -o overwrites.
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	defer file.Close()

	dataWriter := bufio.NewWriter(file)

	_, err = dataWriter.WriteString(string(data))
	if err != nil {
		return err
	}

	color.Greenp("Saved to ", fileName, "\n")
	dataWriter.Flush()

	return nil
}
