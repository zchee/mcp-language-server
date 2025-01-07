package tools

import (
	"fmt"
)

func GetDefinition(symbolName string) (string, error) {
	return fmt.Sprintf("Hello, %s", symbolName), nil
}
