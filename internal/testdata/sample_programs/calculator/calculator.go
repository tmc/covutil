// Package calculator provides basic arithmetic operations
package calculator

import (
	"errors"
	"fmt"
	"math"
)

// Calculator represents a basic calculator
type Calculator struct {
	memory float64
	history []string
}

// New creates a new calculator instance
func New() *Calculator {
	return &Calculator{
		memory:  0,
		history: make([]string, 0),
	}
}

// Add performs addition
func (c *Calculator) Add(a, b float64) float64 {
	result := a + b
	c.recordOperation(fmt.Sprintf("%.2f + %.2f = %.2f", a, b, result))
	return result
}

// Subtract performs subtraction
func (c *Calculator) Subtract(a, b float64) float64 {
	result := a - b
	c.recordOperation(fmt.Sprintf("%.2f - %.2f = %.2f", a, b, result))
	return result
}

// Multiply performs multiplication
func (c *Calculator) Multiply(a, b float64) float64 {
	result := a * b
	c.recordOperation(fmt.Sprintf("%.2f * %.2f = %.2f", a, b, result))
	return result
}

// Divide performs division
func (c *Calculator) Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	result := a / b
	c.recordOperation(fmt.Sprintf("%.2f / %.2f = %.2f", a, b, result))
	return result, nil
}

// Power calculates a to the power of b
func (c *Calculator) Power(a, b float64) float64 {
	result := math.Pow(a, b)
	c.recordOperation(fmt.Sprintf("%.2f ^ %.2f = %.2f", a, b, result))
	return result
}

// Sqrt calculates square root
func (c *Calculator) Sqrt(a float64) (float64, error) {
	if a < 0 {
		return 0, errors.New("square root of negative number")
	}
	result := math.Sqrt(a)
	c.recordOperation(fmt.Sprintf("sqrt(%.2f) = %.2f", a, result))
	return result, nil
}

// StoreMemory stores a value in memory
func (c *Calculator) StoreMemory(value float64) {
	c.memory = value
	c.recordOperation(fmt.Sprintf("M = %.2f", value))
}

// RecallMemory recalls the value from memory
func (c *Calculator) RecallMemory() float64 {
	c.recordOperation(fmt.Sprintf("Recalled M = %.2f", c.memory))
	return c.memory
}

// ClearMemory clears the memory
func (c *Calculator) ClearMemory() {
	c.memory = 0
	c.recordOperation("Memory cleared")
}

// GetHistory returns the calculation history
func (c *Calculator) GetHistory() []string {
	return c.history
}

// ClearHistory clears the calculation history
func (c *Calculator) ClearHistory() {
	c.history = make([]string, 0)
}

// recordOperation adds an operation to the history
func (c *Calculator) recordOperation(operation string) {
	c.history = append(c.history, operation)
	if len(c.history) > 100 {
		// Keep only last 100 operations
		c.history = c.history[1:]
	}
}

// Factorial calculates factorial (not covered by tests initially)
func (c *Calculator) Factorial(n int) (int, error) {
	if n < 0 {
		return 0, errors.New("factorial of negative number")
	}
	if n == 0 || n == 1 {
		return 1, nil
	}
	
	result := 1
	for i := 2; i <= n; i++ {
		result *= i
	}
	c.recordOperation(fmt.Sprintf("%d! = %d", n, result))
	return result, nil
}

// Fibonacci calculates nth Fibonacci number (partially tested)
func (c *Calculator) Fibonacci(n int) (int, error) {
	if n < 0 {
		return 0, errors.New("fibonacci of negative number")
	}
	if n == 0 {
		return 0, nil
	}
	if n == 1 {
		return 1, nil
	}
	
	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	c.recordOperation(fmt.Sprintf("fib(%d) = %d", n, b))
	return b, nil
}