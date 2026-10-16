package browser

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// mockTermPage implements termPage interface for testing multi-term selection
type mockTermPage struct {
	elementCalls        []string
	clickCalls          []string
	elementsResults     map[string]rod.Elements
	elementsErrors      map[string]error
	waitStableErr       error
	waitErr             error
	waitStableCallCount int
	waitCallCount       int
	waitStableFunc      func(timeout time.Duration) error
	elementDelay        time.Duration
	simulateNavigation  bool
	onTimetable         bool
	htmlCalls           int
	htmlContent         string
	htmlErrOnCall       int
	contexts            []context.Context
	navigateErr         error
	navigateCallCount   int
	clickErrors         map[string]error
	changeTermFailAfter int
	changeTermCallCount int
}

func newMockTermPage() *mockTermPage {
	return &mockTermPage{
		elementCalls:    []string{},
		clickCalls:      []string{},
		elementsResults: map[string]rod.Elements{},
		elementsErrors:  map[string]error{},
		clickErrors:     map[string]error{},
	}
}

func (m *mockTermPage) Context(ctx context.Context) termPage {
	m.contexts = append(m.contexts, ctx)
	return m
}

func (m *mockTermPage) Navigate(url string) error {
	m.navigateCallCount++
	return m.navigateErr
}

func (m *mockTermPage) Click(selector string, index int) error {
	call := fmt.Sprintf("%s[%d]", selector, index)
	m.clickCalls = append(m.clickCalls, call)
	if m.elementDelay > 0 {
		time.Sleep(m.elementDelay)
	}
	if m.simulateNavigation {
		switch selector {
		case "input.PSRADIOBUTTON":
			if m.onTimetable {
				return errors.New("term radio buttons are not present on the timetable page")
			}
		case "input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']":
			if !m.onTimetable {
				return errors.New("change term button is not present on the selection page")
			}
		}
	}
	if selector == "input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']" {
		m.changeTermCallCount++
		if m.changeTermFailAfter > 0 && m.changeTermCallCount > m.changeTermFailAfter {
			return errors.New("change term failed after threshold")
		}
	}
	if err, ok := m.clickErrors[call]; ok {
		return err
	}
	if err, ok := m.elementsErrors[selector]; ok {
		return err
	}
	elements, ok := m.elementsResults[selector]
	if !ok || index < 0 || index >= len(elements) {
		return errors.New("element not found")
	}
	if m.simulateNavigation {
		switch selector {
		case "input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']":
			m.onTimetable = false
		case "input[name='DERIVED_SSS_SCT_SSR_PB_GO']":
			m.onTimetable = true
		}
	}
	return nil
}

func (m *mockTermPage) Element(selector string) (*rod.Element, error) {
	m.elementCalls = append(m.elementCalls, selector)
	if m.elementDelay > 0 {
		time.Sleep(m.elementDelay)
	}

	// Check clickErrors first (for simulating element lookup failures)
	if err, ok := m.clickErrors[selector]; ok {
		return nil, err
	}

	if err, ok := m.elementsErrors[selector]; ok {
		return nil, err
	}

	// Handle CSS #id selectors for radio buttons - fall back to input.PSRADIOBUTTON elements
	if strings.HasPrefix(selector, "#SSR_DUMMY_RECV1") {
		if results, ok := m.elementsResults["input.PSRADIOBUTTON"]; ok && len(results) > 0 {
			return results[0], nil
		}
		return nil, errors.New("element not found")
	}

	// Track change term link calls and fail after threshold
	if selector == "input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']" {
		m.changeTermCallCount++
		if m.changeTermFailAfter > 0 && m.changeTermCallCount > m.changeTermFailAfter {
			return nil, errors.New("change term failed after threshold")
		}
	}

	// Handle CSS #id selectors - try both formats
	if strings.HasPrefix(selector, "#") {
		id := selector[1:]
		if results, ok := m.elementsResults["input[name='"+id+"']"]; ok && len(results) > 0 {
			return results[0], nil
		}
	}

	if results, ok := m.elementsResults[selector]; ok && len(results) > 0 {
		return results[0], nil
	}
	return nil, errors.New("element not found")
}

func (m *mockTermPage) Elements(selector string) (rod.Elements, error) {
	if m.elementDelay > 0 {
		time.Sleep(m.elementDelay)
	}
	if m.simulateNavigation {
		if selector == "input.PSRADIOBUTTON" && m.onTimetable {
			return rod.Elements{}, nil
		}
		if selector == "input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']" && !m.onTimetable {
			return rod.Elements{}, nil
		}
	}
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

func (m *mockTermPage) HTML() (string, error) {
	m.htmlCalls++
	if m.htmlContent != "" {
		return m.htmlContent, nil
	}
	if m.htmlErrOnCall > 0 && m.htmlCalls == m.htmlErrOnCall {
		return "", errors.New("synthetic HTML extraction failure")
	}
	return `<!DOCTYPE html><html><body><div id='win0divSTDNT_ENRL_SSV2$0'><div id='win0divDERIVED_REGFRM1_DESCR20$0'><table><tr><td class='PAGROUPDIVIDER' align='left'>MOD1001 - Sample Module</td></tr><tr><td><div id='win0divCLASS_MTG_VW$0'><table cellspacing='0' class='PSLEVEL3GRIDWBO' id='CLASS_MTG_VW$scroll$0'><tr><td><table border='0' cellpadding='2' cellspacing='0' cols='7' width='100%' class='PSLEVEL3GRID'><tr><th scope='col' abbr='Class Nbr' width='29' align='CENTER' class='PSLEVEL3GRIDCOLUMNHDR PSGRIDFIRSTCOLUMN'>Class Nbr</th><th scope='col' abbr='Section' width='26' align='CENTER' class='PSLEVEL3GRIDCOLUMNHDR'>Section</th><th scope='col' abbr='Component' width='51' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Component</th><th scope='col' abbr='Days &amp; Times' width='90' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Days &amp; Times</th><th scope='col' abbr='Room' width='75' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Room</th><th scope='col' abbr='Instructor' width='87' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Instructor</th><th scope='col' abbr='Start/End Date' width='80' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Start/End Date</th></tr><tr id='trCLASS_MTG_VW$0_row1' valign='center'><td><DIV id='win0divDERIVED_CLS_DTL_CLASS_NBR$0'><span class='PSEDITBOX_DISPONLY' id='DERIVED_CLS_DTL_CLASS_NBR$0'>1160</span></DIV></td><td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_SECTION$0'><span id='MTG_SECTION$span$0' class='PSHYPERLINK'><a name='MTG_SECTION$0' id='MTG_SECTION$0'>P1</a></span></DIV></td><td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_COMP$0'><span class='PSEDITBOX_DISPONLY' id='MTG_COMP$0'>Laboratory</span></DIV></td><td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_SCHED$0'><span class='PSEDITBOX_DISPONLY' id='MTG_SCHED$0'>Th 9:00AM - 11:00AM</span></DIV></td><td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_LOC$0'><span class='PSEDITBOX_DISPONLY' id='MTG_LOC$0'>W1-06-18</span></DIV></td><td class='PSLEVEL2GRIDODDROW'><DIV id='win0divDERIVED_CLS_DTL_SSR_INSTR_LONG$0'><span class='PSLONGEDITBOX' id='DERIVED_CLS_DTL_SSR_INSTR_LONG$0'>JOHN DOE</span></DIV></td><td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_DATES$0'><span class='PSEDITBOX_DISPONLY' id='MTG_DATES$0'>17/09/2026 - 17/09/2026</span></DIV></td></tr></table></td></tr></table></div></div></td></tr></table></div></body></html>`, nil
}

func (m *mockTermPage) WaitNavigation(event proto.PageLifecycleEventName) func() {
	return func() {}
}

func testConfig() BrowserConfig {
	cfg := DefaultBrowserConfig(BrowserModeAuto)
	cfg.MaxRetries = 1
	cfg.RetryInterval = 1 * time.Millisecond
	return cfg
}

func TestNavigateToTimetable_StabilityFailureReturnsError(t *testing.T) {
	mock := newMockTermPage()
	mock.waitStableErr = errors.New("page did not stabilize")
	cfg := testConfig()
	cfg.NavigationTimeout = time.Second

	err := navigateToTimetable(context.Background(), mock, cfg)
	if err == nil {
		t.Fatal("expected page stability failure to propagate")
	}
	if !strings.Contains(err.Error(), "wait for timetable page stability") {
		t.Errorf("unexpected stability error: %v", err)
	}
}

func TestFetchTimetablePage_NavigationTimeoutIsScopedToInitialNavigation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cfg := testConfig()
	cfg.NavigationTimeout = 100 * time.Millisecond

	mock := newMockTermPage()
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']"] = rod.Elements{&rod.Element{}}
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{&rod.Element{}}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{&rod.Element{}}
	// Make term selection and fetch work take longer than the initial navigation
	// timeout. They should still run under the caller's timetable-fetch context.
	mock.elementDelay = 120 * time.Millisecond

	entries, err := fetchTimetablePage(ctx, mock, cfg)
	if err != nil {
		t.Fatalf("fetchTimetablePage failed after initial navigation succeeded: %v", err)
	}
	if mock.navigateCallCount != 1 {
		t.Errorf("expected one initial timetable navigation, got %d", mock.navigateCallCount)
	}
	if len(mock.contexts) < 2 {
		t.Fatalf("expected overall and navigation contexts to be applied, got %d", len(mock.contexts))
	}
	parentDeadline, parentHasDeadline := mock.contexts[0].Deadline()
	navDeadline, navHasDeadline := mock.contexts[1].Deadline()
	if !parentHasDeadline || !navHasDeadline {
		t.Fatalf("expected both the overall fetch and initial navigation to have deadlines")
	}
	if !navDeadline.Before(parentDeadline) {
		t.Errorf("expected the initial navigation deadline to expire before the overall fetch deadline, parent=%v navigation=%v", parentDeadline, navDeadline)
	}
	if len(entries) == 0 {
		t.Fatal("expected timetable entries to be fetched from the current page")
	}
}

func TestHandleMultiTermSelection_PropagatesTimetableHTMLFailureAfterRecovery(t *testing.T) {
	ctx := context.Background()
	mock := newMockTermPage()
	mock.simulateNavigation = true
	mock.onTimetable = true
	mock.htmlErrOnCall = 1
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']"] = rod.Elements{&rod.Element{}}
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{&rod.Element{}, &rod.Element{}}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{&rod.Element{}}

	_, err := handleMultiTermSelection(ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected the timetable run to fail after a term's HTML extraction failed")
	}
	if !strings.Contains(err.Error(), "get page HTML for term") {
		t.Errorf("expected HTML extraction error, got: %v", err)
	}
	if mock.htmlCalls != 2 {
		t.Errorf("expected HTML extraction to be attempted for both terms, got %d attempts", mock.htmlCalls)
	}
	if mock.changeTermCallCount != 2 {
		t.Errorf("expected selector recovery after the failed first term, got %d Change Term clicks", mock.changeTermCallCount)
	}
	if len(mock.clickCalls) < 4 ||
		mock.clickCalls[1] != "input.PSRADIOBUTTON[0]" ||
		mock.clickCalls[len(mock.clickCalls)-2] != "input.PSRADIOBUTTON[1]" {
		t.Errorf("expected both term radios to be clicked after recovery, calls: %v", mock.clickCalls)
	}
}

func TestHandleMultiTermSelection_ContextExpired(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mock := newMockTermPage()

	_, err := handleMultiTermSelection(ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected error for expired context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestHandleMultiTermSelection_NoChangeTermButton_FetchCurrentPage(t *testing.T) {
	ctx := context.Background()

	mock := newMockTermPage()
	// No change term button and no radio buttons
	_, err := handleMultiTermSelection(ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should have tried to find change term button
	foundChangeTerm := false
	for _, call := range mock.clickCalls {
		if strings.Contains(call, "DERIVED_SSS_SCT_SSS_TERM_LINK") {
			foundChangeTerm = true
			break
		}
	}
	if !foundChangeTerm {
		t.Error("expected change term button lookup")
	}
}

func TestHandleMultiTermSelection_NoChangeTermButton_EmptyCurrentPageFails(t *testing.T) {
	ctx := context.Background()
	mock := newMockTermPage()
	mock.htmlContent = "<html><body>PeopleSoft error page</body></html>"

	_, err := handleMultiTermSelection(ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected empty current-page parse to fail the timetable fetch")
	}
}

func TestHandleMultiTermSelection_NoRadioButtons_FetchCurrentPage(t *testing.T) {
	ctx := context.Background()

	changeTermEl := &rod.Element{}
	mock := newMockTermPage()
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']"] = rod.Elements{changeTermEl}
	// No radio buttons after clicking change term
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{}

	_, err := handleMultiTermSelection(ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleMultiTermSelection_SingleTerm(t *testing.T) {
	ctx := context.Background()

	changeTermEl := &rod.Element{}
	radioEl := &rod.Element{}
	goEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']"] = rod.Elements{changeTermEl}
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{goEl}

	_, err := handleMultiTermSelection(ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify radio button was clicked (via CSS selector)
	foundRadio := false
	for _, call := range mock.clickCalls {
		if strings.HasPrefix(call, "input.PSRADIOBUTTON[") {
			foundRadio = true
			break
		}
	}
	if !foundRadio {
		t.Error("expected radio button CSS selector lookup")
	}

	// Verify Go button was clicked
	foundGo := false
	for _, call := range mock.clickCalls {
		if strings.Contains(call, "DERIVED_SSS_SCT_SSR_PB_GO") {
			foundGo = true
			break
		}
	}
	if !foundGo {
		t.Error("expected Go button lookup")
	}
}

func TestHandleMultiTermSelection_MultipleTerms(t *testing.T) {
	ctx := context.Background()

	changeTermEl := &rod.Element{}
	radio1 := &rod.Element{}
	radio2 := &rod.Element{}
	goEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']"] = rod.Elements{changeTermEl}
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radio1, radio2}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{goEl}

	_, err := handleMultiTermSelection(ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should have clicked change term twice (once to open, once to return after term 1)
	changeTermCount := 0
	for _, call := range mock.clickCalls {
		if strings.Contains(call, "DERIVED_SSS_SCT_SSS_TERM_LINK") {
			changeTermCount++
		}
	}
	if changeTermCount != 2 {
		t.Errorf("expected 2 change term clicks, got %d", changeTermCount)
	}

	// Should have clicked both radio buttons
	radioCSSCount := 0
	for _, call := range mock.clickCalls {
		if strings.HasPrefix(call, "input.PSRADIOBUTTON[") {
			radioCSSCount++
		}
	}
	if radioCSSCount != 2 {
		t.Errorf("expected 2 radio button clicks, got %d", radioCSSCount)
	}
}

func TestHandleMultiTermSelection_PerTermFailureMarksRunFailed(t *testing.T) {
	ctx := context.Background()

	changeTermEl := &rod.Element{}
	radio1 := &rod.Element{}
	radio2 := &rod.Element{}
	goEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']"] = rod.Elements{changeTermEl}
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radio1, radio2}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{goEl}

	// First radio button click fails
	mock.clickErrors["input.PSRADIOBUTTON[0]"] = errors.New("click failed")

	_, err := handleMultiTermSelection(ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected a term failure to mark the complete timetable fetch as failed")
	}
}

func TestHandleMultiTermSelection_ReturnToTermSelectionFailure_StopsIteration(t *testing.T) {
	ctx := context.Background()

	changeTermEl := &rod.Element{}
	radio1 := &rod.Element{}
	radio2 := &rod.Element{}
	goEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']"] = rod.Elements{changeTermEl}
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radio1, radio2}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{goEl}
	// Fail on second change term lookup (after first term fetch)
	mock.changeTermFailAfter = 1

	_, err := handleMultiTermSelection(ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected failure to return to term selection to mark the whole fetch as failed")
	}

	// Should have clicked only 1 radio button (second was skipped due to return failure)
	radioCSSCount := 0
	for _, call := range mock.clickCalls {
		if strings.HasPrefix(call, "input.PSRADIOBUTTON[") {
			radioCSSCount++
		}
	}
	if radioCSSCount != 1 {
		t.Errorf("expected 1 radio button click after return failure, got %d", radioCSSCount)
	}
}

func TestHandleMultiTermSelection_WrapperWithSuccess(t *testing.T) {
	ctx := context.Background()

	changeTermEl := &rod.Element{}
	radioEl := &rod.Element{}
	goEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']"] = rod.Elements{changeTermEl}
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{goEl}

	_, err := handleMultiTermSelection(ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error from handleMultiTermSelection: %v", err)
	}

	// Verify radio button lookup occurred
	foundRadio := false
	for _, call := range mock.clickCalls {
		if strings.HasPrefix(call, "input.PSRADIOBUTTON[") {
			foundRadio = true
			break
		}
	}
	if !foundRadio {
		t.Error("expected radio button lookup via handleMultiTermSelection")
	}
}

func TestHandleMultiTermSelection_WrapperContextExpired(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mock := newMockTermPage()

	_, err := handleMultiTermSelection(ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected error from handleMultiTermSelection with expired context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

func TestHandleMultiTermSelection_DebugConfig(t *testing.T) {
	ctx := context.Background()

	radioEl := &rod.Element{}
	goEl := &rod.Element{}

	cfg := testConfig()
	cfg.Debug = true

	mock := newMockTermPage()
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSR_PB_GO']"] = rod.Elements{goEl}

	_, err := handleMultiTermSelection(ctx, mock, cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReturnToTermSelection_Success(t *testing.T) {
	ctx := context.Background()

	changeTermEl := &rod.Element{}
	radioEl := &rod.Element{}

	mock := newMockTermPage()
	mock.elementsResults["input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']"] = rod.Elements{changeTermEl}
	mock.elementsResults["input.PSRADIOBUTTON"] = rod.Elements{radioEl}

	err := returnToTermSelection(ctx, mock, testConfig())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify change term button was clicked
	foundChangeTerm := false
	for _, call := range mock.clickCalls {
		if strings.Contains(call, "DERIVED_SSS_SCT_SSS_TERM_LINK") {
			foundChangeTerm = true
			break
		}
	}
	if !foundChangeTerm {
		t.Error("expected change term button click")
	}
}

func TestReturnToTermSelection_ChangeTermNotFound(t *testing.T) {
	ctx := context.Background()

	mock := newMockTermPage()
	// No change term button
	mock.elementsErrors["input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']"] = errors.New("not found")

	err := returnToTermSelection(ctx, mock, testConfig())
	if err == nil {
		t.Fatal("expected error when change term button not found")
	}
	if !strings.Contains(err.Error(), "change term button") {
		t.Errorf("expected 'change term button' in error, got: %v", err)
	}
}

func TestClickElement_NilElementReturnsError(t *testing.T) {
	var nilEl *rod.Element
	if err := clickElement(nilEl); err == nil {
		t.Fatal("expected error when clicking a nil element")
	}
}

func TestClickElement_UninitializedElementReturnsError(t *testing.T) {
	el := &rod.Element{}
	if err := clickElement(el); err == nil {
		t.Fatal("expected an error from an uninitialized Rod element")
	}
}

func TestExtractPageText_NilPage(t *testing.T) {
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
	called := false
	safeRod(func() {
		called = true
	})
	if !called {
		t.Error("safeRod did not execute the function")
	}
}

func TestSafeRod_NilFunction(t *testing.T) {
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
	cfg := testConfig()
	cfg.Debug = true

	debug(cfg, "test message %s %d", "hello", 42)
	debug(cfg, "message without args")
}

func TestDebug_Disabled(t *testing.T) {
	cfg := testConfig()
	cfg.Debug = false

	debug(cfg, "this should not be logged: %s", "test")
}

func TestDebug_EmptyMessage(t *testing.T) {
	cfg := testConfig()
	cfg.Debug = true
	debug(cfg, "")
}
