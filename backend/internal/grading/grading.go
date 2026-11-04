// Package grading holds the university's double-blind marking rule in one place.
//
// Policy: two evaluators independently mark each script. If the gap between the
// two marks is >= Threshold (13 marks) a third external examiner marks it too.
// Below the threshold the two marks are averaged. With three marks the final
// score is the median, which is the mark that survives screening of an outlier.
package grading

import (
	"errors"
	"math"
	"sort"
)

// Threshold is the gap at or above which a third examiner is mandated.
const Threshold = 13.0

var (
	ErrInvalidScore = errors.New("marks must be between 0 and the paper maximum")
	ErrSlotFilled   = errors.New("that evaluator slot is already marked")
)

// Slot identifies an evaluator position on a script.
type Slot string

const (
	SlotFirst  Slot = "FIRST"
	SlotSecond Slot = "SECOND"
	SlotThird  Slot = "THIRD"
)

// Decision is what the rule engine says to do after a submission.
type Decision struct {
	Final     *float64 // set when the script can be resolved
	Method    string   // AVERAGE_OF_TWO | MEDIAN_OF_THREE | ""
	NeedThird bool
	Gap       *float64
}

// Round2 rounds to two decimals, half away from zero, matching the numeric(5,2)
// storage type so the value we decide on is exactly the value we persist.
func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// Evaluate applies the policy to the marks recorded so far.
// marks must be ordered FIRST, SECOND[, THIRD] with no gaps.
func Evaluate(marks []float64, maxMarks float64) (Decision, error) {
	if len(marks) == 0 {
		return Decision{}, errors.New("no marks submitted")
	}
	for _, m := range marks {
		if math.IsNaN(m) || m < 0 || m > maxMarks {
			return Decision{}, ErrInvalidScore
		}
	}

	switch len(marks) {
	case 1:
		return Decision{}, nil
	case 2:
		gap := Round2(math.Abs(marks[0] - marks[1]))
		if gap >= Threshold {
			return Decision{NeedThird: true, Gap: &gap}, nil
		}
		final := Round2((marks[0] + marks[1]) / 2)
		return Decision{Final: &final, Method: "AVERAGE_OF_TWO", Gap: &gap}, nil
	case 3:
		gap2 := Round2(math.Abs(marks[0] - marks[1]))
		sorted := append([]float64(nil), marks...)
		sort.Float64s(sorted)
		final := Round2(sorted[1])
		return Decision{Final: &final, Method: "MEDIAN_OF_THREE", Gap: &gap2}, nil
	default:
		return Decision{}, errors.New("at most three marks are accepted per script")
	}
}

// GradeLetter maps a final mark to the university's classification band.
func GradeLetter(score float64) string {
	switch {
	case score >= 70:
		return "A"
	case score >= 60:
		return "B"
	case score >= 50:
		return "C"
	case score >= 45:
		return "D"
	case score >= 40:
		return "E"
	default:
		return "F"
	}
}
