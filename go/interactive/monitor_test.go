package interactive

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestMonitorRendersFramesUntilEscape(t *testing.T) {
	t.Parallel()

	frames := make(chan []MonitorLine, 2)
	frames <- []MonitorLine{{Text: "Market monitor"}, {Text: "BTC/USDC 100", Style: MessageStyleSuccess}}
	frames <- []MonitorLine{{Text: "Market monitor"}, {Text: "BTC/USDC 101", Style: MessageStyleSuccess}}
	input := bytes.NewBuffer([]byte{27})
	output := &bytes.Buffer{}
	monitor, err := NewMonitor(input, output)
	if err != nil {
		t.Fatalf("NewMonitor() error = %v", err)
	}
	if err := monitor.Run(context.Background(), frames); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if output.String() != "" && !strings.Contains(output.String(), "Market monitor") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestMonitorValidatesInputs(t *testing.T) {
	t.Parallel()

	if _, err := NewMonitor(nil, &bytes.Buffer{}); err == nil {
		t.Fatal("NewMonitor(nil, output) error = nil")
	}
	monitor, _ := NewMonitor(bytes.NewBuffer(nil), &bytes.Buffer{})
	if err := monitor.Run(nil, make(chan []MonitorLine)); err == nil {
		t.Fatal("Run(nil, frames) error = nil")
	}
	if err := monitor.Run(context.Background(), nil); err == nil {
		t.Fatal("Run(ctx, nil) error = nil")
	}
}
