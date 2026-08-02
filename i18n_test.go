package main

import (
	"testing"
)

func TestTranslateSpecialCharacters(t *testing.T) {
	// Garante que o mapa foi inicializado caso não tenha sido na suite principal
	loadLocales()

	// Lista de casos de teste com caracteres especiais
	tests := []struct {
		name     string
		lang     string
		key      string
		expected string
	}{
		{
			name:     "Caracteres especiais SQL",
			lang:     "pt",
			key:      "test'; DROP TABLE users; --",
			expected: "test'; DROP TABLE users; --", // Retorna a própria chave pois não existe
		},
		{
			name:     "Caracteres especiais XSS (HTML/JS)",
			lang:     "en",
			key:      "<script>alert('xss')</script>",
			expected: "<script>alert('xss')</script>",
		},
		{
			name:     "Caracteres unicode/emojis",
			lang:     "es",
			key:      "🦄 ñ ç á é î ö ú",
			expected: "🦄 ñ ç á é î ö ú",
		},
		{
			name:     "Caracteres de formatação e controle",
			lang:     "pt",
			key:      "linha1\nlinha2\t\r",
			expected: "linha1\nlinha2\t\r",
		},
		{
			name:     "Chave válida existente",
			lang:     "pt",
			key:      "title",
			expected: "Canivete da Mata", // Retorna o valor traduzido do pt.json
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translate(tt.lang, tt.key)
			if result != tt.expected {
				t.Errorf("translate(%q, %q) = %q; esperado %q", tt.lang, tt.key, result, tt.expected)
			}
		})
	}
}
