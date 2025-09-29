package util

import "regexp"

// ValidateString validates if string is valid and there is no space in it
func ValidateString(name string) bool {
	stringPattern := regexp.MustCompile("^[A-Za-z .'-]+$")
	return stringPattern.MatchString(name)
}

// Allowed contact numbers -> 9883443344, 09883443344, 0919883443344, +919883443344.....
func ValidateContact(contact string) bool {
	contactPattern := regexp.MustCompile(`^(?:(?:\+|0{0,2})91(\s*[\-]\s*)?|[0]?)?\d{10}$`)
	return contactPattern.MatchString(contact)
}

// ValidateEmail validates email which should be of the type example@domain.com
func ValidateEmail(email string) bool {
	emailPattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-z]{2,}`)
	return emailPattern.MatchString(email)
}
