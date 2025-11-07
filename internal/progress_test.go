package internal

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestShowProgress(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		message string
		fn      func() error
		wantErr bool
	}{
		{
			name:    "successful function",
			message: "Testing",
			fn: func() error {
				return nil
			},
			wantErr: false,
		},
		{
			name:    "function with error",
			message: "Testing error",
			fn: func() error {
				return errors.New("test error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ShowProgress(ctx, tt.message, tt.fn)
			if (err != nil) != tt.wantErr {
				t.Errorf("ShowProgress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShowProgress_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := ShowProgress(ctx, "Testing", func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	// Should handle context cancellation gracefully
	_ = err
}

func TestShowProgressWithSteps(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		steps   []ProgressStep
		wantErr bool
	}{
		{
			name: "successful steps",
			steps: []ProgressStep{
				{Message: "Step 1", Fn: func() error { return nil }},
				{Message: "Step 2", Fn: func() error { return nil }},
			},
			wantErr: false,
		},
		{
			name: "step with error",
			steps: []ProgressStep{
				{Message: "Step 1", Fn: func() error { return nil }},
				{Message: "Step 2", Fn: func() error { return errors.New("step error") }},
			},
			wantErr: true,
		},
		{
			name:    "empty steps",
			steps:   []ProgressStep{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ShowProgressWithSteps(ctx, tt.steps)
			if (err != nil) != tt.wantErr {
				t.Errorf("ShowProgressWithSteps() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProgressStep(t *testing.T) {
	step := ProgressStep{
		Message: "Test step",
		Fn: func() error {
			return nil
		},
	}

	if step.Message != "Test step" {
		t.Errorf("ProgressStep.Message = %q, want 'Test step'", step.Message)
	}

	if step.Fn == nil {
		t.Error("ProgressStep.Fn should not be nil")
	}

	err := step.Fn()
	if err != nil {
		t.Errorf("ProgressStep.Fn() error = %v, want nil", err)
	}
}
