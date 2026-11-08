package premium_level_2

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

var ErrInvalidIntake = errors.New("invalid premium level 2 intake")

// ValidationError carries per-field messages. Field names are server-owned
// catalog keys, never client-supplied paths.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "invalid premium level 2 intake answers" }
func (e *ValidationError) Unwrap() error { return ErrInvalidIntake }

// NormalizeAnswers validates raw client input against the server-owned
// catalog and returns a canonical JSON object containing only known keys with
// normalised values. Unknown keys, malformed types and out-of-range values are
// rejected rather than silently dropped.
func NormalizeAnswers(raw []byte) ([]byte, map[string]string) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, map[string]string{"_form": "Answers are required."}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var input map[string]any
	if err := decoder.Decode(&input); err != nil {
		return nil, map[string]string{"_form": "Answers must be a JSON object."}
	}
	if input == nil {
		return nil, map[string]string{"_form": "Answers must be a JSON object."}
	}
	questions := questionByKey(IntakeCatalog())
	errorsByField := map[string]string{}
	normalized := map[string]any{}

	for key := range input {
		if _, ok := questions[key]; !ok {
			errorsByField[key] = "Unknown field."
		}
	}
	for key, question := range questions {
		value, present := input[key]
		if !present || value == nil {
			if question.Required {
				errorsByField[key] = "This field is required."
			}
			continue
		}
		normalizedValue, message := normalizeAnswer(question, value)
		if message != "" {
			errorsByField[key] = message
			continue
		}
		if normalizedValue != nil {
			normalized[key] = normalizedValue
		}
	}
	if len(errorsByField) > 0 {
		return nil, errorsByField
	}
	if err := crossValidate(normalized); err != nil {
		return nil, map[string]string{"preferred_days": err.Error()}
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return nil, map[string]string{"_form": "Answers could not be processed."}
	}
	return encoded, nil
}

func normalizeAnswer(question Question, value any) (any, string) {
	switch question.Type {
	case QuestionTypeText, QuestionTypeTextarea:
		text, ok := value.(string)
		if !ok {
			return nil, "Must be text."
		}
		text = strings.TrimSpace(text)
		if text == "" {
			if question.Required {
				return nil, "This field is required."
			}
			return nil, ""
		}
		maxLength := question.MaxLength
		if maxLength <= 0 {
			maxLength = MaxAnswerLength
		}
		if len([]rune(text)) > maxLength {
			return nil, fmt.Sprintf("Must be at most %d characters.", maxLength)
		}
		return text, ""
	case QuestionTypeNumber:
		number, ok := value.(json.Number)
		if !ok {
			return nil, "Must be a number."
		}
		parsed, err := number.Float64()
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return nil, "Must be a valid number."
		}
		if question.Min != nil && parsed < *question.Min {
			return nil, fmt.Sprintf("Must be at least %s.", formatNumber(*question.Min))
		}
		if question.Max != nil && parsed > *question.Max {
			return nil, fmt.Sprintf("Must be at most %s.", formatNumber(*question.Max))
		}
		return parsed, ""
	case QuestionTypeSelect:
		text, ok := value.(string)
		if !ok {
			return nil, "Must be one of the available options."
		}
		text = strings.TrimSpace(text)
		if !hasOption(question, text) {
			return nil, "Must be one of the available options."
		}
		return text, ""
	case QuestionTypeMultiSelect:
		values, ok := value.([]any)
		if !ok {
			return nil, "Must be a list of options."
		}
		seen := map[string]bool{}
		result := make([]string, 0, len(values))
		for _, entry := range values {
			text, ok := entry.(string)
			if !ok {
				return nil, "Must be a list of options."
			}
			text = strings.TrimSpace(text)
			if !hasOption(question, text) {
				return nil, "Contains an unknown option."
			}
			if !seen[text] {
				seen[text] = true
				result = append(result, text)
			}
		}
		if question.Required && len(result) == 0 {
			return nil, "This field is required."
		}
		if len(result) == 0 {
			return nil, ""
		}
		return result, ""
	case QuestionTypeBoolean:
		boolean, ok := value.(bool)
		if !ok {
			return nil, "Must be true or false."
		}
		return boolean, ""
	default:
		return nil, "Unsupported field."
	}
}

func hasOption(question Question, value string) bool {
	for _, option := range question.Options {
		if option.Value == value {
			return true
		}
	}
	return false
}

func formatNumber(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
}

func crossValidate(answers map[string]any) error {
	days, _ := answers["preferred_days"].([]string)
	if sessions, ok := answers["sessions_per_week"].(float64); ok && len(days) > 0 && int(sessions) > len(days) {
		return fmt.Errorf("Select at least as many training days as weekly sessions.")
	}
	return nil
}
