package security

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
	validate.RegisterValidation("password", validatePassword)
	validate.RegisterValidation("safe_string", validateSafeString)
}

func Validate(i interface{}) error {
	return validate.Struct(i)
}

func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	
	if len(password) < 8 {
		return false
	}
	
	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)
	
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	
	return hasUpper && hasLower && hasNumber && hasSpecial
}

func validateSafeString(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	// Prevenir XSS y SQL injection básico
	dangerous := []string{"<script", "</script>", "javascript:", "onerror=", "onload=", "--", "/*", "*/", "xp_", "sp_", "exec(", "UNION", "SELECT"}
	
	lower := strings.ToLower(str)
	for _, d := range dangerous {
		if strings.Contains(lower, d) {
			return false
		}
	}
	return true
}

// Sanitize limpia strings de caracteres peligrosos
func Sanitize(input string) string {
	// Remover tags HTML
	re := regexp.MustCompile(`<[^>]*>`)
	sanitized := re.ReplaceAllString(input, "")
	
	// Remover caracteres no imprimibles
	sanitized = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, sanitized)
	
	return strings.TrimSpace(sanitized)
}

// ValidarEmail verifica formato de email
func ValidateEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// ValidarUUID verifica formato UUID
func ValidateUUID(id string) bool {
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	return re.MatchString(strings.ToLower(id))
}