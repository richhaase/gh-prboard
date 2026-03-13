package cmd

import (
	"testing"
)

func TestParseNumberSelection_Single(t *testing.T) {
	result, err := ParseNumberSelection("3", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{3}
	assertIntSliceEqual(t, expected, result)
}

func TestParseNumberSelection_CommaSeparated(t *testing.T) {
	result, err := ParseNumberSelection("1,3,5", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 3, 5}
	assertIntSliceEqual(t, expected, result)
}

func TestParseNumberSelection_Range(t *testing.T) {
	result, err := ParseNumberSelection("2-4", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{2, 3, 4}
	assertIntSliceEqual(t, expected, result)
}

func TestParseNumberSelection_Mixed(t *testing.T) {
	result, err := ParseNumberSelection("1-3,7,9", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 2, 3, 7, 9}
	assertIntSliceEqual(t, expected, result)
}

func TestParseNumberSelection_SpaceSeparated(t *testing.T) {
	result, err := ParseNumberSelection("1 3 5", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 3, 5}
	assertIntSliceEqual(t, expected, result)
}

func TestParseNumberSelection_All(t *testing.T) {
	result, err := ParseNumberSelection("all", 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 2, 3, 4}
	assertIntSliceEqual(t, expected, result)
}

func TestParseNumberSelection_OutOfRange(t *testing.T) {
	_, err := ParseNumberSelection("6", 5)
	if err == nil {
		t.Fatal("expected error for out-of-range input")
	}
}

func TestParseNumberSelection_Zero(t *testing.T) {
	_, err := ParseNumberSelection("0", 5)
	if err == nil {
		t.Fatal("expected error for zero input")
	}
}

func TestParseNumberSelection_InvalidText(t *testing.T) {
	_, err := ParseNumberSelection("abc", 5)
	if err == nil {
		t.Fatal("expected error for invalid text input")
	}
}

func TestParseNumberSelection_Deduplicate(t *testing.T) {
	result, err := ParseNumberSelection("1,1,2-3,2", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 2, 3}
	assertIntSliceEqual(t, expected, result)
}

func assertIntSliceEqual(t *testing.T, expected, actual []int) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Fatalf("expected %v, got %v", expected, actual)
		}
	}
}
