package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

func TestRefusalError(t *testing.T) {
	tests := []struct {
		name     string
		category string
		contains []string
	}{
		{
			name:     "no category",
			category: "",
			contains: []string{"declined the request", "rephrasing"},
		},
		{
			name:     "safeguard category is named",
			category: "bio",
			contains: []string{"declined the request", "bio"},
		},
		{
			name:     "reasoning_extraction explains the cause",
			category: "reasoning_extraction",
			contains: []string{"reasoning_extraction", "internal reasoning"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := refusalError(tt.category)
			if !errors.Is(err, errRefused) {
				t.Errorf("refusalError(%q) does not wrap errRefused: %v", tt.category, err)
			}
			for _, want := range tt.contains {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("refusalError(%q) = %q, want it to contain %q", tt.category, err, want)
				}
			}
		})
	}
}

func TestSeparateTurn(t *testing.T) {
	tests := []struct {
		name             string
		output           string
		content          string
		turnStarted      bool
		want             string
		wantStillStarted bool
	}{
		{name: "first turn gets no break", output: "", content: "Hi", turnStarted: true, want: "Hi"},
		{name: "new turn after text", output: "refers to.", content: "Nothing", turnStarted: true, want: "\n\nNothing"},
		{name: "previous output ends with a newline", output: "refers to.\n", content: "Nothing", turnStarted: true, want: "\nNothing"},
		{name: "previous output already ends with a blank line", output: "`</thinking>`\n\n", content: "Nothing", turnStarted: true, want: "Nothing"},
		{name: "output within a turn is left alone", output: "According to ", content: "the doc", turnStarted: false, want: "the doc"},
		{name: "empty output keeps the break pending", output: "refers to.", content: "", turnStarted: true, want: "", wantStillStarted: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Bods{Output: tt.output, turnStarted: tt.turnStarted}
			if got := b.separateTurn(tt.content); got != tt.want {
				t.Errorf("separateTurn(%q) after %q = %q, want %q", tt.content, tt.output, got, tt.want)
			}
			if b.turnStarted != tt.wantStillStarted {
				t.Errorf("turnStarted = %v, want %v", b.turnStarted, tt.wantStillStarted)
			}
		})
	}
}

// fakeResponseStream stands in for Bedrock's event stream reader so the stream handler can be
// driven with canned events. The SDK exposes the Reader field for exactly this.
type fakeResponseStream struct{ events chan types.ResponseStream }

func (f *fakeResponseStream) Events() <-chan types.ResponseStream { return f.events }
func (f *fakeResponseStream) Close() error                        { return nil }
func (f *fakeResponseStream) Err() error                          { return nil }

// Canned stream events. Every index is 0: bods tracks blocks by arrival order, not by index.
const (
	evMessageStart  = `{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"claude-opus-5-5","usage":{"input_tokens":10,"output_tokens":1}}}`
	evTextStart     = `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`
	evThinkingStart = `{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`
	evSignature     = `{"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"sig"}}`
	evBlockStop     = `{"type":"content_block_stop","index":0}`
	evEndTurn       = `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":5}}`
	evMessageStop   = `{"type":"message_stop","amazon-bedrock-invocationMetrics":{"inputTokenCount":10,"outputTokenCount":5,"invocationLatency":1,"firstByteLatency":1}}`
)

func evDelta(deltaType, field, value string) string {
	v, _ := json.Marshal(value)
	return fmt.Sprintf(`{"type":"content_block_delta","index":0,"delta":{"type":%q,%q:%s}}`, deltaType, field, v)
}

func evText(s string) string     { return evDelta("text_delta", "text", s) }
func evThinking(s string) string { return evDelta("thinking_delta", "thinking", s) }

// newStreamTestBods returns a Bods built the way the TUI builds it, with a fresh conversation.
// The deadline turns a stream that never reaches message_stop into a failure rather than a hang.
func newStreamTestBods(t *testing.T, cfg *Config) *Bods {
	t.Helper()
	saved := messages
	t.Cleanup(func() { messages = saved })
	messages = []Message{{Role: MessageRoleUser, Content: []Content{{Type: MessageContentTypeText, Text: "hi"}}}}

	b := initialBodsModel(stderrRenderer(), cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	b.context = &ctx
	return b
}

// streamTurn runs one assistant response through Update and the stream handler it schedules,
// the way the TUI's event loop does, until the stream ends. Each turn is its own stream, as in
// the tool-use loop. It must end in end_turn: a tool_use stop would invoke Bedrock.
func streamTurn(t *testing.T, b *Bods, events ...string) {
	t.Helper()
	ch := make(chan types.ResponseStream, len(events))
	for _, e := range events {
		ch <- &types.ResponseStreamMemberChunk{Value: types.PayloadPart{Bytes: []byte(e)}}
	}
	stream := bedrockruntime.NewInvokeModelWithResponseStreamEventStream(
		func(s *bedrockruntime.InvokeModelWithResponseStreamEventStream) {
			s.Reader = &fakeResponseStream{events: ch}
		})

	// requestState is where the TUI is while a response streams; from startState, Update would
	// also schedule a stdin read.
	b.state = requestState
	_, cmd := b.Update(completionOutput{stream: stream})
	for b.state != doneState {
		if b.Error != nil {
			t.Fatalf("stream handler failed: %v", b.Error)
		}
		if cmd == nil {
			t.Fatal("stream stopped without reaching message_stop")
		}
		_, cmd = b.Update(cmd())
	}
}

func TestReceiveStreamingMessages(t *testing.T) {
	t.Run("text is replayed exactly as streamed", func(t *testing.T) {
		b := newStreamTestBods(t, &Config{})
		streamTurn(t, b, evMessageStart, evTextStart, evText("Hello"), evText(" world"), evBlockStop, evEndTurn, evMessageStop)

		got := messages[len(messages)-1].Content
		if len(got) != 1 || got[0].Text != "Hello world" {
			t.Errorf("assistant content = %+v, want one text block %q with no placeholder padding", got, "Hello world")
		}
	})

	t.Run("text blocks with no text are dropped", func(t *testing.T) {
		b := newStreamTestBods(t, &Config{})
		streamTurn(t, b, evMessageStart,
			evThinkingStart, evSignature, evBlockStop,
			evTextStart, evBlockStop, // empty
			evTextStart, evText("\n\n"), evBlockStop, // whitespace only
			evEndTurn, evMessageStop)

		got := messages[len(messages)-1].Content
		if len(got) != 1 || got[0].Type != MessageContentTypeThinking || got[0].Signature != "sig" {
			t.Errorf("assistant content = %+v, want only the signed thinking block", got)
		}
	})

	t.Run("consecutive turns are separated, blocks within a turn are not", func(t *testing.T) {
		b := newStreamTestBods(t, &Config{})
		streamTurn(t, b, evMessageStart, evTextStart, evText("I'll check the file."), evBlockStop, evEndTurn, evMessageStop)
		// Citations split one answer into several text blocks, which must stay contiguous.
		streamTurn(t, b, evMessageStart,
			evTextStart, evText("# Answer\n\nAccording to the doc, "), evBlockStop,
			evTextStart, evText("the sky is blue."), evBlockStop,
			evEndTurn, evMessageStop)

		want := "I'll check the file.\n\n# Answer\n\nAccording to the doc, the sky is blue."
		if b.Output != want {
			t.Errorf("Output = %q, want %q", b.Output, want)
		}
	})

	t.Run("the break goes ahead of a thinking marker", func(t *testing.T) {
		b := newStreamTestBods(t, &Config{Format: true})
		streamTurn(t, b, evMessageStart, evTextStart, evText("Checking."), evBlockStop, evEndTurn, evMessageStop)
		streamTurn(t, b, evMessageStart,
			evThinkingStart, evThinking("Plan."), evSignature, evBlockStop,
			evTextStart, evText("Done."), evBlockStop,
			evEndTurn, evMessageStop)

		want := "Checking.\n\n`<thinking>` \n\nPlan.\n\n`</thinking>`\n\nDone."
		if b.Output != want {
			t.Errorf("Output = %q, want %q", b.Output, want)
		}
	})
}
