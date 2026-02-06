#!/bin/bash

# Test script for knowledge graph property tests only
# This isolates the knowledge graph tests from other test files with compilation issues

echo "Running Knowledge Graph Property Tests..."
echo "=========================================="

# Run only the knowledge graph property tests
go test -v \
  -run "TestProperty_KnowledgeGraphEntityStorage|TestProperty_SchemaMigrationDataPreservation" \
  github.com/DevAnuragT/context_keeper/internal/services \
  -count=1

exit_code=$?

if [ $exit_code -eq 0 ]; then
  echo ""
  echo "✓ All knowledge graph property tests passed!"
else
  echo ""
  echo "✗ Some knowledge graph property tests failed"
fi

exit $exit_code
