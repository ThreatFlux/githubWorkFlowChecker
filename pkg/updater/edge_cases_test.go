package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseActionReferencesEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantRefs int // number of action references expected
		wantErr  bool
	}{
		{
			name: "nested steps with multiple uses",
			content: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Nested steps
        uses: actions/setup-node@v2
        steps:
          - uses: actions/setup-python@v3`,
			wantRefs: 3,
			wantErr:  false,
		},
		{
			name: "valid yaml with no actions",
			content: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Hello"`,
			wantRefs: 0,
			wantErr:  false,
		},
		{
			name:     "empty file with valid yaml",
			content:  `{}`,
			wantRefs: 0,
			wantErr:  false,
		},
		{
			name:     "mixed line endings",
			content:  "name: Test Workflow\r\non: [push]\njobs:\r\n  test:\n    runs-on: ubuntu-latest\r\n    steps:\r\n      - uses: actions/checkout@v2\n",
			wantRefs: 1,
			wantErr:  false,
		},
		{
			name: "comments in various positions",
			content: `# Header comment
name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      # Before action
      - uses: actions/checkout@v2  # Inline comment
      # Between actions
      - uses: actions/setup-node@v2
      # After actions`,
			wantRefs: 2,
			wantErr:  false,
		},
		{
			name: "action reference with special characters in version",
			content: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2.3.4-beta.1+meta`,
			wantRefs: 1,
			wantErr:  false,
		},
		{
			name: "multiple jobs with same action",
			content: `name: Test Workflow
on: [push]
jobs:
  test1:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
  test2:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
			wantRefs: 2,
			wantErr:  false,
		},
		{
			name: "complex yaml with anchors and aliases",
			content: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps: &steps
      - uses: actions/checkout@v2
  deploy:
    runs-on: ubuntu-latest
    steps: *steps`,
			wantRefs: 2,
			wantErr:  false,
		},
		{
			name: "unicode characters in workflow",
			content: `name: 测试工作流
on: [push]
jobs:
  テスト:
    runs-on: ubuntu-latest
    steps:
      - name: 检查代码
        uses: actions/checkout@v2`,
			wantRefs: 1,
			wantErr:  false,
		},
		{
			name: "minimal valid workflow",
			content: `on: [push]
jobs:
  a:
    runs-on: a
    steps:
      - uses: actions/checkout@v2`,
			wantRefs: 1,
			wantErr:  false,
		},
	}

	scanner := NewScanner()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "workflow-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create test file
			testFile := filepath.Join(tempDir, "workflow.yml")
			err = os.WriteFile(testFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Parse action references
			refs, err := scanner.ParseActionReferences(testFile)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(refs) != tt.wantRefs {
				t.Errorf("Expected %d references, got %d", tt.wantRefs, len(refs))
			}
		})
	}
}

func TestParseNodeEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantRefs int
		wantErr  bool
	}{
		{
			name: "deeply nested uses",
			content: `name: Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: outer
        uses: actions/checkout@v2
        with:
          nested:
            steps:
              - uses: actions/setup-node@v2`,
			wantRefs: 2,
			wantErr:  false,
		},
		{
			name: "uses in matrix",
			content: `name: Test
on: [push]
jobs:
  test:
    strategy:
      matrix:
        action: ['actions/checkout@v2', 'actions/setup-node@v2']
    runs-on: ubuntu-latest
    steps:
      - uses: ${{ matrix.action }}`,
			wantRefs: 1, // Should only count direct uses
			wantErr:  false,
		},
		{
			name: "uses as plain string",
			content: `name: Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Echo action
        run: |
          echo "uses: actions/checkout@v2"`,
			wantRefs: 0,
			wantErr:  false,
		},
		{
			name: "uses in conditional",
			content: `name: Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - if: ${{ github.event_name == 'push' }}
        uses: actions/checkout@v2`,
			wantRefs: 1,
			wantErr:  false,
		},
	}

	scanner := NewScanner()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "workflow-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create test file
			testFile := filepath.Join(tempDir, "workflow.yml")
			err = os.WriteFile(testFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Parse action references
			refs, err := scanner.ParseActionReferences(testFile)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(refs) != tt.wantRefs {
				t.Errorf("Expected %d references, got %d", tt.wantRefs, len(refs))
			}
		})
	}
}
