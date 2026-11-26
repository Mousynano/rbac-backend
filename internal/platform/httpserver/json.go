package httpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// DecodeJSON reads exactly one JSON object, rejecting unknown fields and trailing data.
func DecodeJSON(r *http.Request, dst any) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return ErrUnsupportedMediaType
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return ErrBodyTooLarge
		}
		return fmt.Errorf("read request body: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 || bytes.TrimSpace(data)[0] != '{' {
		return ErrInvalidJSON
	}
	if err := rejectDuplicateKeys(data); err != nil {
		return ErrInvalidJSON
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return ErrInvalidJSON
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return ErrInvalidJSON
	}
	return nil
}

var (
	ErrUnsupportedMediaType = errors.New("unsupported media type")
	ErrBodyTooLarge         = errors.New("request body too large")
	ErrInvalidJSON          = errors.New("invalid JSON request")
)

func rejectDuplicateKeys(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	var value func() error
	value = func() error {
		t, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				keyToken, err := d.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok || seen[key] {
					return ErrInvalidJSON
				}
				seen[key] = true
				if err := value(); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		case '[':
			for d.More() {
				if err := value(); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		default:
			return ErrInvalidJSON
		}
	}
	if err := value(); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrInvalidJSON
	}
	return nil
}

// WriteJSON writes the standard response envelope and disables caching.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data, "meta": map[string]string{"request_id": w.Header().Get("X-Request-ID")}})
}
