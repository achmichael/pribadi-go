package monitor

import (
	"context"
	"math"
	"strings"

	"github.com/achmichael/pribadi-go/internal/domain"
)

type StructuralEvaluator struct{}

func NewStructuralEvaluator() *StructuralEvaluator {
	return &StructuralEvaluator{}
}

func (e *StructuralEvaluator) Evaluate(_ context.Context, task domain.MonitorTask, currentNumeric *float64, currentText string, previousNumeric *float64, _ string) (bool, error) {
	switch task.Operator {
	case domain.OpAbove:
		if currentNumeric == nil {
			return false, nil
		}
		return *currentNumeric > task.ConditionValue, nil

	case domain.OpBelow:
		if currentNumeric == nil {
			return false, nil
		}
		return *currentNumeric < task.ConditionValue, nil

	case domain.OpPercentChange:
		if currentNumeric == nil || previousNumeric == nil || *previousNumeric == 0 {
			return false, nil
		}
		pct := math.Abs((*currentNumeric - *previousNumeric) / *previousNumeric * 100)
		return pct >= task.ConditionValue, nil

	case domain.OpEquals:
		if currentNumeric != nil {
			return *currentNumeric == task.ConditionValue, nil
		}
		return currentText == task.ConditionText, nil

	case domain.OpContains:
		return strings.Contains(strings.ToLower(currentText), strings.ToLower(task.ConditionText)), nil

	default:
		return false, nil
	}
}
