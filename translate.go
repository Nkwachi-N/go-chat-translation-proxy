package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type cacheEntry struct {
	text      string
	createdAt time.Time
}

type Translator struct {
	client   *http.Client
	url      string
	model    string
	cacheTTL time.Duration
	cache    map[string]cacheEntry
	cacheMu  sync.RWMutex
}

func NewTranslator(url string, model string, cacheTTL time.Duration) *Translator {
	return &Translator{
		url:      url,
		model:    model,
		cacheTTL: cacheTTL,
		client:   &http.Client{Timeout: time.Second * 30},
		cache:    make(map[string]cacheEntry),
	}
}

func (t *Translator) generate(prompt string) (string, error) {
	reqBody := map[string]any{
		"model":  t.model,
		"prompt": prompt,
		"stream": false,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := t.client.Post(t.url+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Response, nil
}

func (t *Translator) DetectLanguage(text string) (string, error) {
	prompt := fmt.Sprintf("What language is this text? Reply with ONLY the ISO language code (e.g. en, pt, es, fr): %s", text)
	return t.generate(prompt)
}

func (t *Translator) Translate(text string, fromLanguage string, toLanguage string) (string, error) {
	prompt := fmt.Sprintf("Translate the following text from %s to %s. Return ONLY the translation, nothing else: %s", fromLanguage, toLanguage, text)
	key := fromLanguage + ":" + toLanguage + ":" + text
	t.cacheMu.RLock()
	translated, ok := t.cache[key]
	t.cacheMu.RUnlock()
	expired := time.Since(translated.createdAt) > t.cacheTTL
	if !ok || expired {
		translated, err := t.generate(prompt)
		if err != nil {
			return translated, err
		}
		t.cacheMu.Lock()
		t.cache[key] = cacheEntry{
			text:      translated,
			createdAt: time.Now(),
		}
		t.cacheMu.Unlock()
		return translated, nil
	}

	return translated.text, nil
}
