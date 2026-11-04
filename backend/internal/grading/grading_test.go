package grading

import (
	"math"
	"testing"
)

func TestEvaluateSingleMark(t *testing.T) {
	d, err := Evaluate([]float64{55}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if d.Final != nil || d.NeedThird {
		t.Fatalf("single mark must stay pending, got %+v", d)
	}
}

func TestThresholdBoundaries(t *testing.T) {
	cases := []struct {
		name      string
		m1, m2    float64
		needThird bool
		wantFinal float64
	}{
		{"just below threshold averages", 70, 82.99, false, 76.5},
		{"exactly threshold triggers third", 70, 83, true, 0},
		{"above threshold triggers third", 50, 63.5, true, 0},
		{"identical marks average", 61, 61, false, 61},
		{"zero and twelve", 0, 12, false, 6},
		{"order independent", 83, 70, true, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, err := Evaluate([]float64{c.m1, c.m2}, 100)
			if err != nil {
				t.Fatal(err)
			}
			if d.NeedThird != c.needThird {
				t.Fatalf("NeedThird=%v want %v", d.NeedThird, c.needThird)
			}
			if !c.needThird {
				if d.Final == nil || math.Abs(*d.Final-c.wantFinal) > 1e-9 {
					t.Fatalf("final=%v want %v", d.Final, c.wantFinal)
				}
				if d.Method != "AVERAGE_OF_TWO" {
					t.Fatalf("method=%q", d.Method)
				}
			}
			if c.needThird && d.Final != nil {
				t.Fatal("third-review decision must not fix a final mark")
			}
		})
	}
}

func TestThirdMarkUsesMedian(t *testing.T) {
	// 40 and 90 split the examiners; the third mark of 60 decides.
	d, err := Evaluate([]float64{40, 90, 60}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if d.Final == nil || *d.Final != 60 {
		t.Fatalf("median expected 60, got %v", d.Final)
	}
	if d.Method != "MEDIAN_OF_THREE" {
		t.Fatalf("method=%q", d.Method)
	}

	// Median ignores an outlier in either direction; the middle mark wins.
	for _, c := range []struct {
		marks []float64
		want  float64
	}{{[]float64{10, 80, 85}, 80}, {[]float64{5, 95, 90}, 90}, {[]float64{90, 5, 85}, 85}} {
		d, err := Evaluate(c.marks, 100)
		if err != nil {
			t.Fatal(err)
		}
		if *d.Final != c.want {
			t.Fatalf("%v: want %v got %v", c.marks, c.want, *d.Final)
		}
	}
}

func TestRounding(t *testing.T) {
	d, err := Evaluate([]float64{70.01, 80}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if *d.Final != 75.01 {
		t.Fatalf("want 75.01 got %v", *d.Final)
	}
	if Round2(75.015) != 75.02 {
		t.Fatalf("half-up rounding broken: %v", Round2(75.015))
	}
}

func TestScoreValidation(t *testing.T) {
	for _, marks := range [][]float64{{101, 80}, {-1, 50}, {math.NaN(), 10}} {
		if _, err := Evaluate(marks, 100); err == nil {
			t.Fatalf("expected error for %v", marks)
		}
	}
	if _, err := Evaluate([]float64{120, 130}, 150); err != nil {
		t.Fatalf("max marks other than 100 must be honoured: %v", err)
	}
	if _, err := Evaluate([]float64{1, 2, 3, 4}, 100); err == nil {
		t.Fatal("four marks must be rejected")
	}
}

func TestGradeLetter(t *testing.T) {
	cases := map[float64]string{70: "A", 69.99: "B", 60: "B", 50: "C", 45: "D", 40: "E", 39.99: "F", 0: "F"}
	for score, want := range cases {
		if got := GradeLetter(score); got != want {
			t.Fatalf("%v -> %s want %s", score, got, want)
		}
	}
}
