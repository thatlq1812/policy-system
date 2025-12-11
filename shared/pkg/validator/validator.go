package validator

import (
	"fmt"
	"regexp"
	"unicode"
)

// Regex patterns for validation
var (
	// Vietnamese phone number: 10-11 digits, starts with 0
	PhoneRegex = regexp.MustCompile(`^0[0-9]{9,10}$`)

	// Email: RFC 5322 simplified pattern
	EmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	// Password: at least 8 chars, 1 uppercase, 1 lowercase, 1 digit, 1 special char
	PasswordMinLength    = 8
	PasswordMaxLength    = 128
	PasswordSpecialChars = `!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?`
)

// ValidatePhoneNumber validates Vietnamese phone numbers
// Supports: 10-11 digits starting with 0
// Examples: 0901234567, 0123456789
func ValidatePhoneNumber(phone string) error {
	if phone == "" {
		return fmt.Errorf("phone number is required")
	}

	if !PhoneRegex.MatchString(phone) {
		return fmt.Errorf("invalid phone number format: must be 10-11 digits starting with 0")
	}

	return nil
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}

	if len(email) > 254 {
		return fmt.Errorf("email too long: maximum 254 characters")
	}

	if !EmailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// PasswordStrength represents password validation requirements
type PasswordStrength struct {
	MinLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireDigit   bool
	RequireSpecial bool
}

// DefaultPasswordStrength returns recommended password requirements
func DefaultPasswordStrength() PasswordStrength {
	return PasswordStrength{
		MinLength:      8,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
	}
}

// BasicPasswordStrength returns minimal password requirements (for backward compatibility)
func BasicPasswordStrength() PasswordStrength {
	return PasswordStrength{
		MinLength:      6,
		RequireUpper:   false,
		RequireLower:   false,
		RequireDigit:   false,
		RequireSpecial: false,
	}
}

// ValidatePassword validates password strength based on requirements
func ValidatePassword(password string, strength PasswordStrength) error {
	if password == "" {
		return fmt.Errorf("password is required")
	}

	if len(password) < strength.MinLength {
		return fmt.Errorf("password must be at least %d characters long", strength.MinLength)
	}

	if len(password) > PasswordMaxLength {
		return fmt.Errorf("password too long: maximum %d characters", PasswordMaxLength)
	}

	var (
		hasUpper   = false
		hasLower   = false
		hasDigit   = false
		hasSpecial = false
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if strength.RequireUpper && !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	if strength.RequireLower && !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}

	if strength.RequireDigit && !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}

	if strength.RequireSpecial && !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// ValidatePlatformRole validates platform role enum
func ValidatePlatformRole(role string) error {
	if role == "" {
		return fmt.Errorf("platform role is required")
	}

	switch role {
	case "Client", "Merchant", "Admin":
		return nil
	default:
		return fmt.Errorf("invalid platform role: must be 'Client', 'Merchant', or 'Admin'")
	}
}

// ValidateUserID validates UUID format
func ValidateUserID(userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}

	// UUID regex: 8-4-4-4-12 hex digits
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(userID) {
		return fmt.Errorf("invalid user ID format: must be valid UUID")
	}

	return nil
}
