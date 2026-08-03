package main

import (
	"embed"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
)

//go:embed locales/*.json
var localesFS embed.FS

var (
	i18n            = make(map[string]map[string]string)
	loadLocalesOnce sync.Once
)

func loadLocales() {
	loadLocalesOnce.Do(func() {
		files, err := localesFS.ReadDir("locales")
		if err != nil {
			log.Fatalf("Erro ao ler locales: %v", err)
		}

		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
				lang := strings.TrimSuffix(file.Name(), ".json")
				data, err := localesFS.ReadFile("locales/" + file.Name())
				if err != nil {
					log.Fatalf("Erro ao ler arquivo locale %s: %v", file.Name(), err)
				}

				var translations map[string]string
				if err := json.Unmarshal(data, &translations); err != nil {
					log.Fatalf("Erro JSON locale %s: %v", file.Name(), err)
				}
				i18n[lang] = translations
			}
		}
	})
}

func getLang(r *http.Request) string {
	if c, err := r.Cookie("lang"); err == nil {
		if _, ok := i18n[c.Value]; ok {
			return c.Value
		}
	}
	// Fallback para pt
	return "pt"
}

func translate(lang, key string) string {
	if dict, ok := i18n[lang]; ok {
		if val, ok2 := dict[key]; ok2 {
			return val
		}
	}
	// Fallback to pt if missing key
	if dict, ok := i18n["pt"]; ok {
		if val, ok2 := dict[key]; ok2 {
			return val
		}
	}
	return key
}
