package db

import "encoding/json"

func StringSliceToJSON(s []string) (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "[]", err
	}
	return string(b), nil
}

func JSONToStringSlice(s string) ([]string, error) {
	var result []string
	err := json.Unmarshal([]byte(s), &result)
	if err != nil {
		return []string{}, err
	}
	return result, nil
}
