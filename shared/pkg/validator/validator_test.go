package validator

import (
	"testing"
)

func TestValidatePhoneNumber(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		wantErr bool
	}{
		{"Valid 10 digits", "0901234567", false},
		{"Valid 11 digits", "01234567890", false},
		{"Invalid - too short", "090123456", true},
		{"Invalid - too long", "090123456789", true},
		{"Invalid - not start with 0", "1901234567", true},
		{"Invalid - contains letters", "090123456a", true},
		{"Empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePhoneNumber(tt.phone)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePhoneNumber() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"Valid email", "test@example.com", false},
		{"Valid with subdomain", "user@mail.company.com", false},
		{"Valid with plus", "user+tag@example.com", false},
		{"Invalid - no @", "testexample.com", true},
		{"Invalid - no domain", "test@", true},
		{"Invalid - no TLD", "test@example", true},
		{"Empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	defaultStrength := DefaultPasswordStrength()
	basicStrength := BasicPasswordStrength()

	tests := []struct {
		name     string
		password string
		strength PasswordStrength
		wantErr  bool
	}{
		{"Valid strong password", "Test123!@#", defaultStrength, false},
		{"Valid basic password", "simple", basicStrength, false},
		{"Too short for default", "Test1!", defaultStrength, true},
		{"No uppercase", "test123!@#", defaultStrength, true},
		{"No lowercase", "TEST123!@#", defaultStrength, true},
		{"No digit", "TestTest!@#", defaultStrength, true},
		{"No special char", "Test12345", defaultStrength, true},
		{"Empty", "", defaultStrength, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password, tt.strength)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePlatformRole(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		wantErr bool
	}{
		{"Valid - Client", "Client", false},
		{"Valid - Merchant", "Merchant", false},
		{"Valid - Admin", "Admin", false},
		{"Invalid - lowercase", "client", true},
		{"Invalid - unknown", "SuperUser", true},
		{"Empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePlatformRole(tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePlatformRole() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUserID(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{"Valid UUID", "550e8400-e29b-41d4-a716-446655440000", false},
		{"Invalid - wrong format", "not-a-uuid", true},
		{"Invalid - missing segment", "550e8400-e29b-41d4-a716", true},
		{"Invalid - uppercase", "550E8400-E29B-41D4-A716-446655440000", true},
		{"Empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserID(tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
