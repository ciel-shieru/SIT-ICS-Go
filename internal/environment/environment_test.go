package environment

import "testing"

func TestIsContainerized_DefaultBuild(t *testing.T) {
	resetForTesting()
	defer resetForTesting()

	// In the default build (no -tags container), IsContainerized returns false.
	if IsContainerized() != false {
		t.Errorf("IsContainerized() = true, want false (default build without -tags container)")
	}
}

func TestIsContainerized_Caching(t *testing.T) {
	resetForTesting()
	defer resetForTesting()

	// Call multiple times — result should be identical (cached).
	got1 := IsContainerized()
	got2 := IsContainerized()
	got3 := IsContainerized()

	if got1 != got2 || got2 != got3 {
		t.Errorf("IsContainerized() returned inconsistent values: %v, %v, %v", got1, got2, got3)
	}

	// In the default build, all should be false.
	if got1 != false {
		t.Errorf("IsContainerized() = %v, want false", got1)
	}
}

func TestResetForTesting(t *testing.T) {
	resetForTesting()
	defer resetForTesting()

	// After reset, IsContainerized() should return the default value again.
	result := IsContainerized()
	if result != false {
		t.Errorf("IsContainerized() after reset = %v, want false", result)
	}
}
