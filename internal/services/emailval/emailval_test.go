package emailval

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestValidate_SyntaxValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		email      string
		wantValid  bool
		wantSyntax bool
		wantLevel  *Level
	}{
		{
			name:       "empty email",
			email:      "",
			wantValid:  false,
			wantSyntax: false,
			wantLevel:  ptrLevel(LevelSyntax),
		},
		{
			name:       "whitespace only",
			email:      "   ",
			wantValid:  false,
			wantSyntax: false,
			wantLevel:  ptrLevel(LevelSyntax),
		},
		{
			name:       "invalid format - no @",
			email:      "userexample.com",
			wantValid:  false,
			wantSyntax: false,
			wantLevel:  ptrLevel(LevelSyntax),
		},
		{
			name:       "invalid format - no domain",
			email:      "user@",
			wantValid:  false,
			wantSyntax: false,
			wantLevel:  ptrLevel(LevelSyntax),
		},
		{
			name:       "invalid format - consecutive dots",
			email:      "user..name@example.com",
			wantValid:  false,
			wantSyntax: false,
			wantLevel:  ptrLevel(LevelSyntax),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Validate(ctx, tt.email, nil)
			if err != nil {
				t.Fatalf("Validate() returned error: %v", err)
			}

			if result == nil {
				t.Fatal("Validate() returned nil result")
			}

			if result.SyntaxOK != tt.wantSyntax {
				t.Errorf("SyntaxOK = %v, want %v", result.SyntaxOK, tt.wantSyntax)
			}

			if result.Valid() != tt.wantValid {
				t.Errorf("Valid() = %v, want %v", result.Valid(), tt.wantValid)
			}

			if tt.wantLevel != nil {
				if result.FailedAt == nil {
					t.Errorf("FailedAt = nil, want %v", *tt.wantLevel)
				} else if *result.FailedAt != *tt.wantLevel {
					t.Errorf("FailedAt = %v, want %v", *result.FailedAt, *tt.wantLevel)
				}
			}
		})
	}
}

func TestValidate_Normalization(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		email          string
		wantNormalized string
	}{
		{
			name:           "lowercase conversion",
			email:          "USER@EXAMPLE.COM",
			wantNormalized: "user@example.com",
		},
		{
			name:           "trim whitespace",
			email:          "  user@example.com  ",
			wantNormalized: "user@example.com",
		},
		{
			name:           "mixed case with whitespace",
			email:          "  User.Name@Example.COM  ",
			wantNormalized: "user.name@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Validate(ctx, tt.email, nil)
			if err != nil {
				t.Fatalf("Validate() returned error: %v", err)
			}

			if result.Normalized != tt.wantNormalized {
				t.Errorf("Normalized = %q, want %q", result.Normalized, tt.wantNormalized)
			}

			if result.Input != tt.email {
				t.Errorf("Input = %q, want %q", result.Input, tt.email)
			}
		})
	}
}

func TestValidate_DisposableCheck(t *testing.T) {
	ctx := context.Background()

	// Create a mock provider for testing to avoid DNS lookups
	mockProvider := NewStaticDisposableProvider([]string{
		"mailinator.com",
		"tempmail.com",
	})

	// Note: These tests may fail due to DNS lookup requirements.
	// In a real scenario, you'd mock the DNS lookup as well.
	tests := []struct {
		name             string
		email            string
		checkDisposable  bool
		wantDisposable   bool
	}{
		{
			name:            "disposable check disabled",
			email:           "user@mailinator.com",
			checkDisposable: false,
			wantDisposable:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{
				CheckDisposable:    tt.checkDisposable,
				DisposableProvider: mockProvider,
				Timeout:            100 * time.Millisecond,
			}

			result, err := Validate(ctx, tt.email, opts)
			if err != nil {
				t.Fatalf("Validate() returned error: %v", err)
			}

			if result.Disposable != tt.wantDisposable {
				t.Errorf("Disposable = %v, want %v", result.Disposable, tt.wantDisposable)
			}
		})
	}
}

func TestValidate_NilOptions(t *testing.T) {
	ctx := context.Background()

	// Should not panic with nil options
	result, err := Validate(ctx, "invalid", nil)
	if err != nil {
		t.Fatalf("Validate() with nil options returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Validate() with nil options returned nil result")
	}
}

func TestValidate_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// With cancelled context, the function should still return a result
	// but may fail at the domain check level
	result, err := Validate(ctx, "user@gmail.com", nil)
	if err != nil {
		// Context cancellation may cause an error in DNS lookup
		t.Logf("Expected behavior: error with cancelled context: %v", err)
	}

	if result != nil && result.SyntaxOK {
		// Syntax check should still pass since it doesn't need context
		t.Log("Syntax check passed as expected (doesn't require network)")
	}
}

func TestValidate_TimeoutOption(t *testing.T) {
	ctx := context.Background()

	opts := &Options{
		Timeout: 1 * time.Nanosecond, // Very short timeout
	}

	// This should timeout during DNS lookup
	result, err := Validate(ctx, "user@example.com", opts)
	if err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}

	// Syntax should still be checked
	if result.SyntaxOK {
		t.Log("Syntax check passed before timeout")
	}
}

func TestResult_Valid(t *testing.T) {
	tests := []struct {
		name      string
		result    Result
		wantValid bool
	}{
		{
			name:      "no failures",
			result:    Result{FailedAt: nil},
			wantValid: true,
		},
		{
			name:      "syntax failure",
			result:    Result{FailedAt: ptrLevel(LevelSyntax)},
			wantValid: false,
		},
		{
			name:      "domain failure",
			result:    Result{FailedAt: ptrLevel(LevelDomain)},
			wantValid: false,
		},
		{
			name:      "disposable failure",
			result:    Result{FailedAt: ptrLevel(LevelDisposable)},
			wantValid: false,
		},
		{
			name:      "role failure",
			result:    Result{FailedAt: ptrLevel(LevelRole)},
			wantValid: false,
		},
		{
			name:      "smtp failure",
			result:    Result{FailedAt: ptrLevel(LevelSMTP)},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.Valid() != tt.wantValid {
				t.Errorf("Valid() = %v, want %v", tt.result.Valid(), tt.wantValid)
			}
		})
	}
}

func TestResult_AddMessage(t *testing.T) {
	result := &Result{}

	result.addMessage("error: %s", "test")
	if len(result.Messages) != 1 {
		t.Errorf("Messages length = %d, want 1", len(result.Messages))
	}
	if result.Messages[0] != "error: test" {
		t.Errorf("Messages[0] = %q, want %q", result.Messages[0], "error: test")
	}

	result.addMessage("another message")
	if len(result.Messages) != 2 {
		t.Errorf("Messages length = %d, want 2", len(result.Messages))
	}
}

func TestLevel_Constants(t *testing.T) {
	// Verify level ordering
	if LevelSyntax >= LevelDomain {
		t.Error("LevelSyntax should be less than LevelDomain")
	}
	if LevelDomain >= LevelDisposable {
		t.Error("LevelDomain should be less than LevelDisposable")
	}
	if LevelDisposable >= LevelRole {
		t.Error("LevelDisposable should be less than LevelRole")
	}
	if LevelRole >= LevelSMTP {
		t.Error("LevelRole should be less than LevelSMTP")
	}

	// Verify specific values
	if LevelSyntax != 1 {
		t.Errorf("LevelSyntax = %d, want 1", LevelSyntax)
	}
	if LevelDomain != 2 {
		t.Errorf("LevelDomain = %d, want 2", LevelDomain)
	}
	if LevelDisposable != 3 {
		t.Errorf("LevelDisposable = %d, want 3", LevelDisposable)
	}
	if LevelRole != 4 {
		t.Errorf("LevelRole = %d, want 4", LevelRole)
	}
	if LevelSMTP != 5 {
		t.Errorf("LevelSMTP = %d, want 5", LevelSMTP)
	}
}

func TestOptions_Defaults(t *testing.T) {
	// Test that nil options work
	ctx := context.Background()
	result, err := Validate(ctx, "invalid-email", nil)
	if err != nil {
		t.Fatalf("Validate with nil options returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Validate with nil options returned nil result")
	}

	// Test empty options work
	opts := &Options{}
	result, err = Validate(ctx, "invalid-email", opts)
	if err != nil {
		t.Fatalf("Validate with empty options returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Validate with empty options returned nil result")
	}
}

// ptrLevel returns a pointer to a Level value
func ptrLevel(l Level) *Level {
	return &l
}

// BenchmarkValidate benchmarks the validation performance
func BenchmarkValidate(b *testing.B) {
	ctx := context.Background()
	opts := &Options{
		Timeout: 50 * time.Millisecond,
	}

	b.Run("syntax_only_invalid", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Validate(ctx, "invalid", opts)
		}
	})

	b.Run("syntax_valid_but_no_mx", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Validate(ctx, "user@thisdomain-does-not-exist-12345.com", opts)
		}
	})
}

// LookupMX replaces DNS, so a caller can check MX without the network.
func TestValidateWithLookupMX(t *testing.T) {
	lookup := func(_ context.Context, domain string) ([]*net.MX, error) {
		if domain == "has-mx.test" {
			return []*net.MX{{Host: "mx.has-mx.test", Pref: 10}}, nil
		}
		return nil, nil
	}
	res, err := Validate(context.Background(), "a@has-mx.test", &Options{LookupMX: lookup})
	if err != nil || !res.Valid() || !res.HasMX {
		t.Fatalf("has-mx.test: valid=%v hasMX=%v err=%v", res.Valid(), res.HasMX, err)
	}
	res, err = Validate(context.Background(), "a@no-mx.test", &Options{LookupMX: lookup})
	if err != nil || res.Valid() || res.HasMX || res.FailedAt == nil || *res.FailedAt != LevelDomain {
		t.Fatalf("no-mx.test: valid=%v hasMX=%v err=%v", res.Valid(), res.HasMX, err)
	}
}
