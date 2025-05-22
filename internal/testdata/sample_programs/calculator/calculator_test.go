package calculator

import (
	"testing"
)

func TestBasicOperations(t *testing.T) {
	calc := New()
	
	// Test addition
	result := calc.Add(2, 3)
	if result != 5 {
		t.Errorf("Add(2, 3) = %f; want 5", result)
	}
	
	// Test subtraction
	result = calc.Subtract(5, 3)
	if result != 2 {
		t.Errorf("Subtract(5, 3) = %f; want 2", result)
	}
	
	// Test multiplication
	result = calc.Multiply(4, 3)
	if result != 12 {
		t.Errorf("Multiply(4, 3) = %f; want 12", result)
	}
}

func TestDivision(t *testing.T) {
	calc := New()
	
	// Test normal division
	result, err := calc.Divide(10, 2)
	if err != nil {
		t.Errorf("Divide(10, 2) returned error: %v", err)
	}
	if result != 5 {
		t.Errorf("Divide(10, 2) = %f; want 5", result)
	}
	
	// Test division by zero
	_, err = calc.Divide(10, 0)
	if err == nil {
		t.Error("Divide(10, 0) should return error")
	}
}

func TestPower(t *testing.T) {
	calc := New()
	
	result := calc.Power(2, 3)
	if result != 8 {
		t.Errorf("Power(2, 3) = %f; want 8", result)
	}
}

func TestSqrt(t *testing.T) {
	calc := New()
	
	// Test normal square root
	result, err := calc.Sqrt(9)
	if err != nil {
		t.Errorf("Sqrt(9) returned error: %v", err)
	}
	if result != 3 {
		t.Errorf("Sqrt(9) = %f; want 3", result)
	}
	
	// Test negative number (only sometimes tested)
	if testing.Short() {
		t.Skip("Skipping negative sqrt test in short mode")
	}
	_, err = calc.Sqrt(-1)
	if err == nil {
		t.Error("Sqrt(-1) should return error")
	}
}

func TestMemory(t *testing.T) {
	calc := New()
	
	// Test store and recall
	calc.StoreMemory(42)
	result := calc.RecallMemory()
	if result != 42 {
		t.Errorf("RecallMemory() = %f; want 42", result)
	}
	
	// Test clear memory
	calc.ClearMemory()
	result = calc.RecallMemory()
	if result != 0 {
		t.Errorf("RecallMemory() after clear = %f; want 0", result)
	}
}

func TestHistory(t *testing.T) {
	calc := New()
	
	calc.Add(1, 2)
	calc.Subtract(5, 3)
	
	history := calc.GetHistory()
	if len(history) != 2 {
		t.Errorf("History length = %d; want 2", len(history))
	}
	
	calc.ClearHistory()
	history = calc.GetHistory()
	if len(history) != 0 {
		t.Errorf("History length after clear = %d; want 0", len(history))
	}
}

// This test only runs sometimes (varies coverage)
func TestFibonacci(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Fibonacci test in short mode")
	}
	
	calc := New()
	
	result, err := calc.Fibonacci(5)
	if err != nil {
		t.Errorf("Fibonacci(5) returned error: %v", err)
	}
	if result != 5 {
		t.Errorf("Fibonacci(5) = %d; want 5", result)
	}
	
	// Test edge cases
	result, err = calc.Fibonacci(0)
	if err != nil {
		t.Errorf("Fibonacci(0) returned error: %v", err)
	}
	if result != 0 {
		t.Errorf("Fibonacci(0) = %d; want 0", result)
	}
}

// Factorial is never tested (0% coverage)