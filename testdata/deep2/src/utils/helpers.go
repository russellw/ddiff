package utils

import "strings"

func FormatName(name string) string {
    return strings.Title(strings.ToLower(name))
}

func ValidateEmail(email string) bool {
    return strings.Contains(email, "@")
}