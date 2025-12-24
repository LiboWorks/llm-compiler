package integration

import (
	"fmt"
	"strings"
	"testing"
	"time"

	llmctesting "github.com/LiboWorks/llm-compiler/internal/testing"
)

func TestShellBasicWorkflow(t *testing.T) {
	runner, err := llmctesting.NewTestRunner(t)
	if err != nil {
		t.Fatalf("failed to create test runner: %v", err)
	}

	fixture := runner.GetFixture("shell_basic")
	// Use unique name to avoid conflicts with parallel tests
	fixture.Name = fmt.Sprintf("shell_basic_%d", time.Now().UnixNano())
	result, err := runner.CompileAndRun(fixture, 30*time.Second)
	if err != nil {
		t.Fatalf("CompileAndRun failed: %v", err)
	}

	assertions := llmctesting.NewAssertions(t, result)
	assertions.
		Completed().
		ExitCode(0).
		ContextHasValue("shell_basic", "hello_result", "hello world")
}

func TestCrossWorkflowCommunication(t *testing.T) {
	runner, err := llmctesting.NewTestRunner(t)
	if err != nil {
		t.Fatalf("failed to create test runner: %v", err)
	}

	fixture := runner.GetFixture("cross_workflow")
	// Use unique name to avoid conflicts with parallel tests
	fixture.Name = fmt.Sprintf("cross_workflow_%d", time.Now().UnixNano())
	result, err := runner.CompileAndRun(fixture, 30*time.Second)
	if err != nil {
		t.Fatalf("CompileAndRun failed: %v", err)
	}

	assertions := llmctesting.NewAssertions(t, result)
	assertions.
		Completed().
		ExitCode(0)

	// Verify consumer received the producer's output
	if result.Contexts != nil {
		if consumer, ok := result.Contexts["consumer"]; ok {
			if val, ok := consumer["producer.produce"]; ok {
				if !strings.Contains(val, "hello-from-producer") {
					t.Errorf("expected producer.produce to contain 'hello-from-producer', got %q", val)
				}
			}
		}
	}
}

func TestTemplateRendering(t *testing.T) {
	runner, err := llmctesting.NewTestRunner(t)
	if err != nil {
		t.Fatalf("failed to create test runner: %v", err)
	}

	fixture := runner.GetFixture("template")
	// Use unique name to avoid conflicts with parallel tests
	fixture.Name = fmt.Sprintf("template_%d", time.Now().UnixNano())
	result, err := runner.CompileAndRun(fixture, 30*time.Second)
	if err != nil {
		t.Fatalf("CompileAndRun failed: %v", err)
	}

	assertions := llmctesting.NewAssertions(t, result)
	assertions.
		Completed().
		ExitCode(0)

	// Verify the template message was rendered correctly (echo adds newline)
	if result.Contexts != nil {
		if ctx, ok := result.Contexts["template_test"]; ok {
			if msg, ok := ctx["message"]; ok {
				if !strings.Contains(msg, "Hello") || !strings.Contains(msg, "Alice") {
					t.Errorf("expected message to contain 'Hello' and 'Alice', got %q", msg)
				}
			}
		}
	}
}

func TestParallelWorkflows(t *testing.T) {
	runner, err := llmctesting.NewTestRunner(t)
	if err != nil {
		t.Fatalf("failed to create test runner: %v", err)
	}

	fixture := runner.GetFixture("parallel")
	// Use unique name to avoid conflicts with parallel tests
	fixture.Name = fmt.Sprintf("parallel_%d", time.Now().UnixNano())
	result, err := runner.CompileAndRun(fixture, 30*time.Second)
	if err != nil {
		t.Fatalf("CompileAndRun failed: %v", err)
	}

	assertions := llmctesting.NewAssertions(t, result)
	assertions.
		Completed().
		ExitCode(0)

	// Verify all three workflows completed
	if result.Contexts == nil {
		t.Fatal("no contexts captured")
	}

	for _, name := range []string{"parallel_a", "parallel_b", "parallel_c"} {
		if _, ok := result.Contexts[name]; !ok {
			t.Errorf("workflow %q context not found", name)
		}
	}
}

func TestConditionalExecution(t *testing.T) {
	runner, err := llmctesting.NewTestRunner(t)
	if err != nil {
		t.Fatalf("failed to create test runner: %v", err)
	}

	fixture := runner.GetFixture("conditional")
	// Use unique name to avoid conflicts with parallel tests
	fixture.Name = fmt.Sprintf("conditional_%d", time.Now().UnixNano())
	result, err := runner.CompileAndRun(fixture, 30*time.Second)
	if err != nil {
		t.Fatalf("CompileAndRun failed: %v", err)
	}

	assertions := llmctesting.NewAssertions(t, result)
	assertions.
		Completed().
		ExitCode(0)

	// Conditional that matched should have output
	if result.Contexts != nil {
		ctx := result.Contexts["conditional_test"]
		if ctx != nil {
			if ctx["conditional_result"] == "" {
				t.Error("conditional_result should have value when condition matches")
			}
			// negative_result should be empty because condition didn't match
			if ctx["negative_result"] != "" {
				t.Error("negative_result should be empty when condition doesn't match")
			}
		}
	}
}

// TestAllFixtures runs a basic compile-and-run test on all fixtures
func TestAllFixtures(t *testing.T) {
	runner, err := llmctesting.NewTestRunner(t)
	if err != nil {
		t.Fatalf("failed to create test runner: %v", err)
	}

	fixtures, err := runner.ListFixtures()
	if err != nil {
		t.Fatalf("failed to list fixtures: %v", err)
	}

	if len(fixtures) == 0 {
		t.Fatal("no fixtures found")
	}

	for _, fixture := range fixtures {
		// Skip error_handling as it's designed to fail
		if fixture.Name == "error_handling" {
			continue
		}

		t.Run(fixture.Name, func(t *testing.T) {
			result, err := runner.CompileAndRun(fixture, 30*time.Second)
			if err != nil {
				t.Fatalf("CompileAndRun failed for %s: %v", fixture.Name, err)
			}

			llmctesting.NewAssertions(t, result).Completed()
		})
	}
}
