package browser

import (
	"errors"
	"testing"
)

func TestExtractADFSLoginError_SpanElement(t *testing.T) {
	mockPage := &mockPageHTML{html: `<span id="errorText" for="" aria-live="assertive" role="alert">MFA error message here</span>`}
	err, _ := ExtractADFSLoginError(mockPage)
	if err != "MFA error message here" {
		t.Errorf("expected 'MFA error message here', got '%s'", err)
	}
}

func TestExtractADFSLoginError_CredentialError(t *testing.T) {
	mockPage := &mockPageHTML{html: `<span id="errorText">Incorrect user ID or password. Type the correct user ID and password, and try again.</span>`}
	err, _ := ExtractADFSLoginError(mockPage)
	if err != "Incorrect user ID or password. Type the correct user ID and password, and try again." {
		t.Errorf("expected credential error text, got '%s'", err)
	}
}

func TestExtractADFSLoginError_DivWrapper(t *testing.T) {
	mockPage := &mockPageHTML{html: `<div id="error" class="fieldMargin error smallText"><span id="errorText">Incorrect user ID or password</span></div>`}
	err, _ := ExtractADFSLoginError(mockPage)
	if err != "Incorrect user ID or password" {
		t.Errorf("expected 'Incorrect user ID or password', got '%s'", err)
	}
}

func TestExtractADFSLoginError_NoError(t *testing.T) {
	mockPage := &mockPageHTML{html: `<div id="verificationCodeInput"></div>`}
	err, _ := ExtractADFSLoginError(mockPage)
	if err != "" {
		t.Errorf("expected empty string, got '%s'", err)
	}
}

func TestExtractADFSLoginError_CaseInsensitive(t *testing.T) {
	mockPage := &mockPageHTML{html: `<span id="errorText">INCORRECT USER ID OR PASSWORD</span>`}
	err, _ := ExtractADFSLoginError(mockPage)
	if err != "INCORRECT USER ID OR PASSWORD" {
		t.Errorf("expected uppercase error text, got '%s'", err)
	}
}

func TestExtractADFSLoginError_PElement(t *testing.T) {
	mockPage := &mockPageHTML{html: `<p id="errorText">Legacy p tag error</p>`}
	err, _ := ExtractADFSLoginError(mockPage)
	if err != "Legacy p tag error" {
		t.Errorf("expected 'Legacy p tag error', got '%s'", err)
	}
}

func TestExtractADFSLoginError_EmptyError(t *testing.T) {
	mockPage := &mockPageHTML{html: `<span id="errorText"></span>`}
	err, _ := ExtractADFSLoginError(mockPage)
	if err != "" {
		t.Errorf("expected empty string for empty error element, got '%s'", err)
	}
}

func TestExtractADFSLoginError_NestedText(t *testing.T) {
	mockPage := &mockPageHTML{html: `<span id="errorText"><span>Nested</span> text content</span>`}
	err, _ := ExtractADFSLoginError(mockPage)
	if err != "Nested text content" {
		t.Errorf("expected 'Nested text content', got '%s'", err)
	}
}

func TestHasADFSCredentialError_Detected(t *testing.T) {
	html := `<span id="errorText" for="" aria-live="assertive" role="alert">Incorrect user ID or password. Type the correct user ID and password, and try again.</span>`
	if !hasADFSCredentialError(html) {
		t.Error("expected credential error to be detected")
	}
}

func TestHasADFSCredentialError_DivWrapper(t *testing.T) {
	html := `<div id="error" class="fieldMargin error smallText"><span id="errorText">Incorrect user ID or password. Type the correct user ID and password, and try again.</span></div>`
	if !hasADFSCredentialError(html) {
		t.Error("expected credential error to be detected with div wrapper")
	}
}

func TestHasADFSCredentialError_CaseInsensitive(t *testing.T) {
	html := `<span id="errorText">INCORRECT USER ID OR PASSWORD</span>`
	if !hasADFSCredentialError(html) {
		t.Error("expected case-insensitive match")
	}
}

func TestHasADFSCredentialError_NoErrorElement(t *testing.T) {
	html := `<div id="verificationCodeInput"></div>`
	if hasADFSCredentialError(html) {
		t.Error("expected no credential error")
	}
}

func TestHasADFSCredentialError_DifferentError(t *testing.T) {
	html := `<span id="errorText">Something else entirely</span>`
	if hasADFSCredentialError(html) {
		t.Error("expected no credential error for different message")
	}
}

func TestHasADFSCredentialError_EmptyError(t *testing.T) {
	html := `<span id="errorText"></span>`
	if hasADFSCredentialError(html) {
		t.Error("expected no credential error for empty message")
	}
}

func TestHasADFSCredentialError_NoErrorTextElement(t *testing.T) {
	html := `<div><span id="someOtherId">Incorrect user ID or password</span></div>`
	if hasADFSCredentialError(html) {
		t.Error("expected no credential error for non-errorText element")
	}
}

func TestHasADFSCredentialError_PartialMatch(t *testing.T) {
	html := `<span id="errorText">Your user ID is incorrect</span>`
	if hasADFSCredentialError(html) {
		t.Error("expected no credential error for partial match")
	}
}

func TestIsADFSCredentialError_Detected(t *testing.T) {
	mockPage := &mockPageHTML{html: `<span id="errorText">Incorrect user ID or password</span>`}
	isCredErr, err := IsADFSCredentialError(mockPage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isCredErr {
		t.Error("expected credential error to be detected")
	}
}

func TestIsADFSCredentialError_NotDetected(t *testing.T) {
	mockPage := &mockPageHTML{html: `<div id="verificationCodeInput"></div>`}
	isCredErr, err := IsADFSCredentialError(mockPage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isCredErr {
		t.Error("expected no credential error")
	}
}

func TestIsADFSCredentialError_HTMLExtractionError(t *testing.T) {
	wantErr := errors.New("extraction failed")
	mockPage := &mockPageHTML{html: "", err: wantErr}
	_, err := IsADFSCredentialError(mockPage)
	if err == nil {
		t.Fatal("expected error for HTML extraction failure")
	}
}

type mockPageHTML struct {
	html string
	err  error
}

func (m *mockPageHTML) HTML() (string, error) {
	return m.html, m.err
}
