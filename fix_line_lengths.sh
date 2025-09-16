#!/bin/bash

# Script to automatically fix common line length issues
# Focuses on MarkdownDescription strings and function signatures

echo "Fixing line length issues..."

# Pattern 1: Long MarkdownDescription strings
find internal/provider -name "*.go" -exec sed -i '' \
  -e 's/MarkdownDescription: "\([^"]*\)\. \([^"]*\)"/MarkdownDescription: "\1. " +\
					"\2"/g' \
  -e 's/MarkdownDescription: "\([^"]*\) Valid values: \([^"]*\)"/MarkdownDescription: "\1 " +\
					"Valid values: \2"/g' \
  {} \;

# Pattern 2: Long function signatures - break after context.Context
find internal/provider -name "*.go" -exec sed -i '' \
  -e 's/func (\([^)]*\)) \([A-Za-z]*\)(ctx context\.Context, \([^,]*\), \([^)]*\)) {/func (\1) \2(\
		ctx context.Context, \3, \4) {/g' \
  {} \;

# Pattern 3: Long function calls with multiple parameters
find internal/provider -name "*.go" -exec sed -i '' \
  -e 's/\(.*Executor\.Run(ctx, [^,]*, \)\[\([^]]*\)\], \(.*\)/\1\
		[\2], \3/g' \
  {} \;

echo "Line length fixes applied. Run golangci-lint to check results."
