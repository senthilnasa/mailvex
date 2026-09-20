package template

import (
	"strings"
	"testing"
)

func TestMustacheTemplateRendering(t *testing.T) {
	tmpl := "Hello {{name}}! Welcome to {{company.name}}. Your code is {{code}}."
	vars := map[string]interface{}{
		"name": "Alice",
		"company": map[string]interface{}{
			"name": "Acme Corp",
		},
		"code": 12345,
	}

	res, err := Render(tmpl, vars, Options{StrictVariables: true, AutoEscape: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Hello Alice! Welcome to Acme Corp. Your code is 12345."
	if res != expected {
		t.Fatalf("expected '%s', got '%s'", expected, res)
	}
}

func TestTemplateHTMLAutoEscape(t *testing.T) {
	tmpl := "<p>Hello {{name}}</p><div>Raw: {{{raw_html}}}</div>"
	vars := map[string]interface{}{
		"name":     "<script>alert('xss')</script>",
		"raw_html": "<strong>Bold</strong>",
	}

	res, err := Render(tmpl, vars, Options{AutoEscape: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(res, "<script>") {
		t.Fatalf("expected script tag to be escaped, got: %s", res)
	}
	if !strings.Contains(res, "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;") {
		t.Fatalf("expected escaped script tag, got: %s", res)
	}
	if !strings.Contains(res, "<strong>Bold</strong>") {
		t.Fatalf("expected raw html to be unescaped, got: %s", res)
	}
}

func TestTemplateSections(t *testing.T) {
	tmpl := "Items:{{#items}}\n- {{name}}: ${{price}}{{/items}}"
	vars := map[string]interface{}{
		"items": []map[string]interface{}{
			{"name": "Apples", "price": "3.50"},
			{"name": "Oranges", "price": "4.00"},
		},
	}

	res, err := Render(tmpl, vars, Options{AutoEscape: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(res, "- Apples: $3.50") || !strings.Contains(res, "- Oranges: $4.00") {
		t.Fatalf("expected rendered items section, got: %s", res)
	}
}

func TestTemplateMissingVariable(t *testing.T) {
	tmpl := "Hello {{name}}, order #{{order_id}}"
	vars := map[string]interface{}{
		"name": "Bob",
	}

	// Non-strict should render empty
	res, err := Render(tmpl, vars, Options{StrictVariables: false})
	if err != nil {
		t.Fatalf("unexpected error in non-strict: %v", err)
	}
	if res != "Hello Bob, order #" {
		t.Fatalf("expected 'Hello Bob, order #', got '%s'", res)
	}

	// Strict should return error
	_, err = Render(tmpl, vars, Options{StrictVariables: true})
	if err == nil || !strings.Contains(err.Error(), "template_variable_missing") {
		t.Fatalf("expected template_variable_missing error, got: %v", err)
	}
}
