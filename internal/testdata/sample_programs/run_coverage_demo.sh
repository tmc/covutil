#!/bin/bash

# Coverage demonstration script for covtree and covforest

set -e

echo "🧪 Coverage Demo: Multiple test runs with varying patterns"
echo "========================================================="

# Create coverage directories
mkdir -p coverage_runs/run1
mkdir -p coverage_runs/run2
mkdir -p coverage_runs/run3
mkdir -p coverage_runs/run4

echo ""
echo "📊 Run 1: Basic tests only (Calculator)"
echo "----------------------------------------"
cd calculator
GOCOVERDIR=../coverage_runs/run1 go test -cover -short ./...
cd ..

echo ""
echo "📊 Run 2: Full tests (Calculator with Fibonacci)"
echo "------------------------------------------------"
cd calculator
GOCOVERDIR=../coverage_runs/run2 go test -cover ./...
cd ..

echo ""
echo "📊 Run 3: String utils basic tests"
echo "-----------------------------------"
cd stringutils
GOCOVERDIR=../coverage_runs/run3 go test -cover -short ./...
cd ..

echo ""
echo "📊 Run 4: String utils with full coverage"
echo "------------------------------------------"
cd stringutils
FULL_TEST=1 GOCOVERDIR=../coverage_runs/run4 go test -cover ./...
cd ..

echo ""
echo "🌳 Analyzing coverage with covtree"
echo "=================================="

echo ""
echo "🔍 Debug: What coverage directories do we have?"
covtree debug -i=coverage_runs

echo ""
echo "📈 Coverage percentages by package:"
covtree percent -i=coverage_runs

echo ""
echo "🔧 Coverage by function:"
covtree func -i=coverage_runs

echo ""
echo "📦 Packages with coverage data:"
covtree pkglist -i=coverage_runs

echo ""
echo "📝 JSON output sample:"
covtree json -i=coverage_runs | head -10

echo ""
echo "🌲 Building coverage forest"
echo "============================"

echo ""
echo "➕ Adding trees to forest..."
covforest add -i=coverage_runs/run1 -name="Calculator Basic Tests" -machine="demo-machine" -branch="main" -repo="covutil-demo"
covforest add -i=coverage_runs/run2 -name="Calculator Full Tests" -machine="demo-machine" -branch="main" -repo="covutil-demo"
covforest add -i=coverage_runs/run3 -name="StringUtils Basic Tests" -machine="demo-machine" -branch="feature-strings" -repo="covutil-demo"
covforest add -i=coverage_runs/run4 -name="StringUtils Full Tests" -machine="demo-machine" -branch="feature-strings" -repo="covutil-demo"

echo ""
echo "📋 Forest summary:"
covforest summary

echo ""
echo "📑 List all trees:"
covforest list

echo ""
echo "🎯 Coverage comparison across runs:"
covforest list -format=csv | head -5

echo ""
echo "🚀 Starting web server for interactive exploration..."
echo "     Visit http://localhost:8080 to explore the coverage forest"
echo "     Press Ctrl+C to stop the server"

covforest serve -http=:8080