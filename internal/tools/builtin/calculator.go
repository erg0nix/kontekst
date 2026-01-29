package builtin

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/erg0nix/kontekst/internal/tools"
)

type Calculator struct{}

func (tool *Calculator) Name() string        { return "calculator" }
func (tool *Calculator) Description() string { return "Performs basic arithmetic" }
func (tool *Calculator) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"op": map[string]any{"type": "string"},
			"a":  map[string]any{"type": "number"},
			"b":  map[string]any{"type": "number"},
		},
		"required": []string{"op", "a", "b"},
	}
}
func (tool *Calculator) RequiresApproval() bool { return false }

func (tool *Calculator) Execute(args map[string]any, _ context.Context) (string, error) {
	rawOp, _ := args["op"].(string)
	leftOperand, _ := args["a"].(float64)
	rightOperand, _ := args["b"].(float64)
	normalizedOp := strings.TrimSpace(rawOp)

	switch normalizedOp {
	case "+":
		normalizedOp = "add"
	case "-":
		normalizedOp = "sub"
	case "*", "x":
		normalizedOp = "mul"
	case "/":
		normalizedOp = "div"
	case "^":
		normalizedOp = "pow"
	}

	switch normalizedOp {
	case "add":
		return fmt.Sprintf("%g", leftOperand+rightOperand), nil
	case "sub":
		return fmt.Sprintf("%g", leftOperand-rightOperand), nil
	case "mul":
		return fmt.Sprintf("%g", leftOperand*rightOperand), nil
	case "div":
		if rightOperand == 0 {
			return "", errors.New("division by zero")
		}
		return fmt.Sprintf("%g", leftOperand/rightOperand), nil
	case "pow":
		return fmt.Sprintf("%g", math.Pow(leftOperand, rightOperand)), nil
	default:
		return "", errors.New("unknown operation")
	}
}

func RegisterCalculator(registry *tools.Registry) {
	registry.Add(&Calculator{})
}
