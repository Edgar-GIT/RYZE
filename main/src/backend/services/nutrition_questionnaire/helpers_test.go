package nutrition_questionnaire

// Pointer helpers for building test submissions. They live in the test package
// so the production contract surface stays free of unused symbols.
func StringPtr(v string) *string { return &v }

func IntPtr(v int) *int { return &v }

func FloatPtr(v float64) *float64 { return &v }

func StringsPtr(v ...string) *[]string { return &v }
