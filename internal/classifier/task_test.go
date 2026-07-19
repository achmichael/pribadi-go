package classifier

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
)

func TestTaskClassifier_NoActiveTask(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	c := &taskClassifier{logger: &logger}

	result, err := c.Classify(nil, TaskClassifyParams{
		UserText:   "Apa itu neural network?",
		ActiveTask: "",
		TurnCount:  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ContinuesTask {
		t.Error("expected continues_task=false when no active task")
	}
	if result.NewTaskDesc == "" {
		t.Error("expected new task description")
	}
}

func TestTaskClassifier_FirstTurn(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	c := &taskClassifier{logger: &logger}

	result, err := c.Classify(nil, TaskClassifyParams{
		UserText:   "Summarize this document",
		ActiveTask: "old task",
		TurnCount:  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ContinuesTask {
		t.Error("expected continues_task=false on first turn")
	}
}

func TestTaskClassifier_ShortFollowup(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	c := &taskClassifier{logger: &logger}

	result, err := c.Classify(nil, TaskClassifyParams{
		UserText:   "lanjutkan",
		ActiveTask: "summarize document",
		TurnCount:  3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ContinuesTask {
		t.Error("expected continues_task=true for short follow-up")
	}
}

func TestSummarizeTask(t *testing.T) {
	c := &taskClassifier{}

	// Short text
	result := c.summarizeTask("short text")
	if result != "short text" {
		t.Errorf("expected unchanged short text, got %s", result)
	}

	// Long text
	long := "This is a very long text that should be truncated because it exceeds the maximum allowed length for a task description summary field"
	result = c.summarizeTask(long)
	if len(result) > 90 { // 80 + "..."
		t.Errorf("expected truncated text, got len=%d", len(result))
	}
}
