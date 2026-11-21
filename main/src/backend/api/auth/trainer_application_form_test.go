package auth_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"testing"

	"github.com/google/uuid"
)

// validApplicationMultipart builds a complete trainer application: the form
// fields, a certification (PDF) and an identification (PDF), each unique so
// the duplicate-document check never links two test accounts. extra adds
// other fields (for example to prove client-sent identities are ignored).
func validApplicationMultipart(t testing.TB, extra map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	work, _ := json.Marshal([]map[string]any{{"organization": "Ginásio Central", "role": "Personal trainer", "start_year": 2018, "end_year": nil}})
	education, _ := json.Marshal([]map[string]any{{"institution": "Faculdade de Motricidade Humana", "qualification": "Licenciatura em Desporto", "year": 2017}})
	fields := map[string]string{
		"full_name":        "Rita Costa",
		"birth_date":       "1992-04-15",
		"city":             "Lisboa",
		"profession":       "Personal trainer",
		"years_experience": "6",
		"work_history":     string(work),
		"education":        string(education),
		"about":            "Treino força e condição física há seis anos, com foco em iniciantes e reabilitação de lesões ligeiras.",
	}
	for key, value := range extra {
		fields[key] = value
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	for _, kind := range []string{"certification", "identification"} {
		part, err := writer.CreateFormFile(kind, kind+".pdf")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write([]byte("%PDF-1.4\n% " + uuid.NewString() + "\n1 0 obj << /Producer (Scanner) >>\nendobj\n%%EOF\n"))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body, writer.FormDataContentType()
}
