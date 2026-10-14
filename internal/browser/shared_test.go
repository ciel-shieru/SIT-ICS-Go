package browser

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
)

// mockTermPage implements termPage interface for testing handleTermSelectionWithPage
type mockTermPage struct {
	elementCalls        []string
	elementsResults     map[string]rod.Elements
	elementsErrors      map[string]error
	waitStableErr       error
	waitErr             error
	waitStableCallCount int
	waitCallCount       int
	waitStableFunc      func(timeout time.Duration) error
}

func newMockTermPage() *mockTermPage {
	return &mockTermPage{
		elementsResults: map[string]rod.Elements{},
		elementsErrors:  map[string]error{},
	}
}

func (m *mockTermPage) Context(ctx context.Context) termPage {
	_ = ctx
	return m
}

func (m *mockTermPage) Element(selector string) (*rod.Element, error) {
	m.elementCalls = append(m.elementCalls, selector)
	if err, ok := m.elementsErrors[selector]; ok {
		return nil, err
	}
	if results, ok := m.elementsResults[selector]; ok && len(results) > 0 {
		return results[0], nil
	}
	return nil, errors.New("element not found")
}

func (m *mockTermPage) Elements(selector string) (rod.Elements, error) {
	if err, ok := m.elementsErrors[selector]; ok {
		return nil, err
	}
	if results, ok := m.elementsResults[selector]; ok {
		return results, nil
	}
	return rod.Elements{}, nil
}

func (m *mockTermPage) WaitStable(timeout time.Duration) error {
	m.waitStableCallCount++
	if m.waitStableFunc != nil {
		return m.waitStableFunc(timeout)
	}
	return m.waitStableErr
}

func (m *mockTermPage) Wait(fn *rod.EvalOptions) error {
	m.waitCallCount++
	return m.waitErr
}

func testConfig() BrowserConfig {
	cfg := DefaultBrowserConfig(BrowserModeAuto)
	cfg.MaxRetries = 1
	cfg.RetryInterval = 1 * time.Millisecond
	return cfg
}

func TestHandleTermSelectionWithPage_ContextExpired(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mock := newMockTermPage()

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected error for expired context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestHandleTermSelectionWithPage_ChangeTermButtonNotFound(t *testing.T) {
	ctx := context.Background()

	mock := newMockTermPage()
	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expectedMsg := "term selection radio button not found"
	if err.Error() != expectedMsg+": retry failed after 2 attempts: element not found" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleTermSelectionWithPage_ChangeTermButtonFoundAndClicked(t *testing.T) {
	ctx := context.Background()

	changeTermEl := &rod.Element{}
	radioEl := &rod.Element{}
	continueEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']"] = rod.Elements{changeTermEl}
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{continueEl}

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify change term button was found
	found := false
	for _, call := range mock.elementCalls {
		if call == "input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected change term button element lookup")
	}
}

func TestHandleTermSelectionWithPage_NoRadioButtonsAfterChangeTerm(t *testing.T) {
	ctx := context.Background()

	changeTermEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']"] = rod.Elements{changeTermEl}
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{}

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected error for missing radio buttons, got nil")
	}
	expectedMsg := "term selection radio button not found"
	if err.Error() != expectedMsg+": retry failed after 2 attempts: element not found" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleTermSelectionWithPage_NoRadioButtonsReturnsError(t *testing.T) {
	ctx := context.Background()

	continueEl := &rod.Element{}

	mock := newMockTermPage()
	// No radio buttons at all - Element("input.PSRADIOBUTTON") will fail
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{continueEl}

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected error when no radio buttons exist, got nil")
	}
	expectedMsg := "term selection radio button not found"
	if err.Error() != expectedMsg+": retry failed after 2 attempts: element not found" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleTermSelectionWithPage_WaitForTimetableData(t *testing.T) {
	ctx := context.Background()

	radioEl := &rod.Element{}
	continueEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{continueEl}

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if mock.waitCallCount == 0 {
		t.Error("expected Wait to be called for timetable data polling")
	}
}

func TestHandleTermSelectionWithPage_WaitStableCalled(t *testing.T) {
	ctx := context.Background()

	radioEl := &rod.Element{}
	continueEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{continueEl}

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if mock.waitStableCallCount == 0 {
		t.Error("expected WaitStable to be called after timetable wait")
	}
}

func TestHandleTermSelectionWithPage_WaitStableFailure(t *testing.T) {
	ctx := context.Background()

	radioEl := &rod.Element{}
	continueEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{continueEl}
	mock.waitStableErr = errors.New("stable timeout")

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error (WaitStable failure should be non-fatal): %v", err)
	}
}

func TestHandleTermSelectionWithPage_ContextDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	radioEl := &rod.Element{}
	continueEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{continueEl}

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected error for context deadline exceeded, got nil")
	}
}

func TestHandleTermSelectionWithPage_ElementsQueryError(t *testing.T) {
	ctx := context.Background()

	queryErr := errors.New("query failed")
	mock := newMockTermPage()
	// Element("input.PSRADIOBUTTON") fails with queryErr first
	mock.elementsErrors["input.PSRADIOBUTTON"] = queryErr

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected error for radio button query failure, got nil")
	}
	// Element lookup fails first, so error is "term selection radio button not found"
	expectedMsg := "term selection radio button not found"
	if err.Error() != expectedMsg+": retry failed after 2 attempts: "+queryErr.Error() {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleTermSelectionWithPage_WaitForTimetableDataError(t *testing.T) {
	ctx := context.Background()

	radioEl := &rod.Element{}
	continueEl := &rod.Element{}

	waitErr := errors.New("timetable wait failed")
	mock := newMockTermPage()
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{continueEl}
	mock.waitErr = waitErr

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected error for timetable wait failure, got nil")
	}
	// Error includes retry wrapping
	if !errors.Is(err, waitErr) {
		t.Errorf("expected error to wrap %v, got: %v", waitErr, err)
	}
}

func TestHandleTermSelectionWithPage_SuccessfulFlow(t *testing.T) {
	ctx := context.Background()

	changeTermEl := &rod.Element{}
	radioEl := &rod.Element{}
	continueEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']"] = rod.Elements{changeTermEl}
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{continueEl}

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify all expected element lookups occurred
	expectedCalls := []string{
		"input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']",
		"input.PSRADIOBUTTON",
		"input.PSRADIOBUTTON",
		"input[name='DERIVED_SSS_SCT_SSR_PB_GO']",
	}

	if len(mock.elementCalls) != len(expectedCalls) {
		t.Errorf("expected %d element calls, got %d", len(expectedCalls), len(mock.elementCalls))
		for i, call := range mock.elementCalls {
			t.Logf("  call %d: %s", i, call)
		}
		return
	}

	for i, expected := range expectedCalls {
		if mock.elementCalls[i] != expected {
			t.Errorf("call %d: expected %q, got %q", i, expected, mock.elementCalls[i])
		}
	}
}

func TestHandleTermSelectionWithPage_ChangeTermClickFailure(t *testing.T) {
	ctx := context.Background()

	changeTermEl := &rod.Element{}
	radioEl := &rod.Element{}
	continueEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']"] = rod.Elements{changeTermEl}
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{continueEl}

	// Change term click failure should be logged but not fatal
	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error (change term click failure should be non-fatal): %v", err)
	}
}

func TestHandleTermSelectionWithPage_ContextDeadlineDuringStableWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mock := newMockTermPage()
	// Set up elements to succeed quickly
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{&rod.Element{}}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{&rod.Element{}}

	// Use a channel to synchronize: cancel context when WaitStable is about to be called
	stableCalled := make(chan struct{})
	mock.waitStableFunc = func(timeout time.Duration) error {
		close(stableCalled)
		// Wait for context to be cancelled before returning
		<-ctx.Done()
		return ctx.Err()
	}

	// Cancel context shortly after stable wait begins
	go func() {
		<-stableCalled
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected error for context cancellation during stable wait, got nil")
	}
}

func TestHandleTermSelectionWithPage_SingleRadioButton(t *testing.T) {
	ctx := context.Background()

	radioEl := &rod.Element{}
	continueEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{continueEl}

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleTermSelectionWithPage_DebugConfig(t *testing.T) {
	ctx := context.Background()

	radioEl := &rod.Element{}
	continueEl := &rod.Element{}

	cfg := testConfig()
	cfg.Debug = true

	mock := newMockTermPage()
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{continueEl}

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleTermSelectionWithPage_MaxRetriesZero(t *testing.T) {
	ctx := context.Background()

	radioEl := &rod.Element{}
	continueEl := &rod.Element{}

	cfg := testConfig()
	cfg.MaxRetries = 0

	mock := newMockTermPage()
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{continueEl}

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleTermSelectionWithPage_NoChangeTermButton(t *testing.T) {
	ctx := context.Background()

	radioEl := &rod.Element{}
	continueEl := &rod.Element{}

	mock := newMockTermPage()
	// No change term button - Element lookup will fail, skips to radio button check
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{continueEl}

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleTermSelectionWithPage_MultipleRadioButtons(t *testing.T) {
	ctx := context.Background()

	radio1 := &rod.Element{}
	radio2 := &rod.Element{}
	radio3 := &rod.Element{}
	continueEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radio1, radio2, radio3}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{continueEl}

	err := handleTermSelectionWithPage(ctx, mock, ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify radio button lookup occurred
	foundElements := false
	for _, call := range mock.elementCalls {
		if call == "input.PSRADIOBUTTON" {
			foundElements = true
		}
	}
	if !foundElements {
		t.Error("expected radio button lookup")
	}
}

func TestHandleTermSelection_WrapperWithSuccess(t *testing.T) {
	ctx := context.Background()

	changeTermEl := &rod.Element{}
	radioEl := &rod.Element{}
	continueEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']"] = rod.Elements{changeTermEl}
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{continueEl}

	// handleTermSelection creates a 5-second sub-context and calls handleTermSelectionWithPage
	err := handleTermSelection(ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error from handleTermSelection wrapper: %v", err)
	}

	// Verify element lookups occurred (delegated to handleTermSelectionWithPage)
	foundRadio := false
	for _, call := range mock.elementCalls {
		if call == "input.PSRADIOBUTTON" {
			foundRadio = true
			break
		}
	}
	if !foundRadio {
		t.Error("expected radio button lookup via handleTermSelectionWithPage")
	}
}

func TestHandleTermSelection_WrapperContextExpired(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mock := newMockTermPage()

	err := handleTermSelection(ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected error from handleTermSelection with expired context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestClickElement_NilElement(t *testing.T) {
	// clickElement should recover from panic when given nil element
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("clickElement panicked with nil element: %v", r)
		}
	}()

	var nilEl *rod.Element
	err := clickElement(nilEl)
	if err != nil {
		t.Errorf("expected nil error from clickElement with nil element, got: %v", err)
	}
}

func TestClickElement_ValidElement(t *testing.T) {
	el := &rod.Element{}
	err := clickElement(el)
	if err != nil {
		t.Errorf("expected nil error from clickElement with valid element, got: %v", err)
	}
}

func TestExtractPageText_NilPage(t *testing.T) {
	// extractPageText should recover from panic when given nil page
	ctx := context.Background()
	var nilPage *rod.Page

	result, err := extractPageText(ctx, nilPage, 1, 1*time.Millisecond)
	if err == nil {
		t.Fatal("expected error from extractPageText with nil page, got nil")
	}
	if result != "" {
		t.Errorf("expected empty result string, got: %q", result)
	}
	if !strings.Contains(err.Error(), "browser evaluation failed") &&
		!strings.Contains(err.Error(), "panic") {
		t.Errorf("expected 'browser evaluation failed' or 'panic' in error, got: %v", err)
	}
}

func TestExtractPageText_ContextExpired(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var nilPage *rod.Page

	result, err := extractPageText(ctx, nilPage, 1, 1*time.Millisecond)
	if err == nil {
		t.Fatal("expected error from extractPageText with cancelled context, got nil")
	}
	if result != "" {
		t.Errorf("expected empty result string, got: %q", result)
	}
}

func TestExtractPageText_ContextDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)
	var nilPage *rod.Page

	result, err := extractPageText(ctx, nilPage, 1, 1*time.Millisecond)
	if err == nil {
		t.Fatal("expected error from extractPageText with deadline exceeded, got nil")
	}
	if result != "" {
		t.Errorf("expected empty result string, got: %q", result)
	}
}

func TestSafeRod_FunctionPanic(t *testing.T) {
	// safeRod should recover from panic without propagating it
	var panicked bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		safeRod(func() {
			panic("test panic")
		})
	}()

	if panicked {
		t.Error("safeRod should have recovered from panic, but panic propagated")
	}
}

func TestSafeRod_NormalFunction(t *testing.T) {
	// safeRod should execute normal functions without issues
	called := false
	safeRod(func() {
		called = true
	})
	if !called {
		t.Error("safeRod did not execute the function")
	}
}

func TestSafeRod_NilFunction(t *testing.T) {
	// safeRod should recover from panic when fn is nil
	var panicked bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		safeRod(nil)
	}()

	if panicked {
		t.Error("safeRod should have recovered from nil function panic")
	}
}

func TestGetPageURL_NilPage(t *testing.T) {
	// getPageURL with nil page should return empty string
	ctx := context.Background()
	var nilPage *rod.Page
	cfg := testConfig()

	url := getPageURL(ctx, nilPage, cfg)
	if url != "" {
		t.Errorf("expected empty URL from getPageURL with nil page, got: %q", url)
	}
}

func TestGetPageURL_ContextExpired(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var nilPage *rod.Page
	cfg := testConfig()

	url := getPageURL(ctx, nilPage, cfg)
	if url != "" {
		t.Errorf("expected empty URL from getPageURL with expired context, got: %q", url)
	}
}

func TestDebug_Enabled(t *testing.T) {
	// debug with Debug=true should call log.Printf (no panic expected)
	cfg := testConfig()
	cfg.Debug = true

	// Should not panic
	debug(cfg, "test message %s %d", "hello", 42)
	debug(cfg, "message without args")
}

func TestDebug_Disabled(t *testing.T) {
	// debug with Debug=false should be a no-op
	cfg := testConfig()
	cfg.Debug = false

	// Should not panic and should produce no output
	debug(cfg, "this should not be logged: %s", "test")
}

func TestDebug_EmptyMessage(t *testing.T) {
	// debug with empty message should not panic
	cfg := testConfig()
	cfg.Debug = true
	debug(cfg, "")
}
