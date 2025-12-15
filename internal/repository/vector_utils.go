package repository

import (
	"fmt"
	"strconv"
	"strings"
)

// VectorToString конвертирует []float32 в строку для pgvector: '[0.1,0.2,...]'
func VectorToString(vec []float32) string {
	if len(vec) == 0 {
		return "[]"
	}

	parts := make([]string, len(vec))
	for i, v := range vec {
		parts[i] = strconv.FormatFloat(float64(v), 'f', -1, 32)
	}

	return "[" + strings.Join(parts, ",") + "]"
}

// StringToVector конвертирует строку pgvector в []float32
func StringToVector(s string) ([]float32, error) {
	// Убираем квадратные скобки
	s = strings.Trim(s, "[]")
	if s == "" {
		return []float32{}, nil
	}

	parts := strings.Split(s, ",")
	vec := make([]float32, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		val, err := strconv.ParseFloat(part, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse vector element %d: %w", i, err)
		}
		vec[i] = float32(val)
	}

	return vec, nil
}

