package agent

import "encoding/json"

func jsonItems(body string) []map[string]any {
	var page struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal([]byte(body), &page); err != nil {
		return nil
	}
	return page.Items
}

func jsonObject(body string) map[string]any {
	var obj map[string]any
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		return nil
	}
	return obj
}

func jsonString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}
