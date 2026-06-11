package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/kingbenny101/kbarr/internal/config"
	"github.com/kingbenny101/kbarr/internal/core/db"
)

func TestKbdex(indexerAddr string) func(context.Context, *struct{}) (*TestResultOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*TestResultOutput, error) {
		resp, err := http.Post(indexerAddr+"/test/kbdex", "application/json", nil)
		if err != nil {
			return &TestResultOutput{Body: TestResult{Ok: "false", Message: err.Error()}}, nil
		}
		defer resp.Body.Close()
		var result TestResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			result = TestResult{Ok: "false", Message: "upstream returned non-JSON"}
		}
		return &TestResultOutput{Body: result}, nil
	}
}

func TestIndexer(indexerAddr string) func(context.Context, *struct{}) (*TestResultOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*TestResultOutput, error) {
		resp, err := http.Post(indexerAddr+"/test/prowlarr", "application/json", nil)
		if err != nil {
			return &TestResultOutput{Body: TestResult{Ok: "false", Message: err.Error()}}, nil
		}
		defer resp.Body.Close()
		var result TestResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			result = TestResult{Ok: "false", Message: "upstream returned non-JSON"}
		}
		return &TestResultOutput{Body: result}, nil
	}
}

func TestDownloader(downloaderAddr string) func(context.Context, *struct{}) (*TestResultOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*TestResultOutput, error) {
		resp, err := http.Post(downloaderAddr+"/test", "application/json", nil)
		if err != nil {
			return &TestResultOutput{Body: TestResult{Ok: "false", Message: err.Error()}}, nil
		}
		defer resp.Body.Close()
		var result TestResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			result = TestResult{Ok: "false", Message: "upstream returned non-JSON"}
		}
		return &TestResultOutput{Body: result}, nil
	}
}

func TriggerDownloader(downloaderAddr string) func(context.Context, *struct{}) (*TestResultOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*TestResultOutput, error) {
		resp, err := http.Post(downloaderAddr+"/trigger", "application/json", nil)
		if err != nil {
			return &TestResultOutput{Body: TestResult{Ok: "false", Message: err.Error()}}, nil
		}
		defer resp.Body.Close()
		var result TestResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			result = TestResult{Ok: "false", Message: "upstream returned non-JSON"}
		}
		return &TestResultOutput{Body: result}, nil
	}
}

func TestJellyfin() func(context.Context, *struct{}) (*TestResultOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*TestResultOutput, error) {
		jellyfinURL := strings.TrimRight(config.Get(db.DB, "jellyfinUrl", ""), "/")
		if jellyfinURL == "" {
			return &TestResultOutput{Body: TestResult{Ok: "false", Message: "jellyfinUrl is not configured"}}, nil
		}
		apiKey := config.Get(db.DB, "jellyfinApiKey", "")

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, jellyfinURL+"/System/Info", nil)
		if err != nil {
			return &TestResultOutput{Body: TestResult{Ok: "false", Message: err.Error()}}, nil
		}
		req.Header.Set("X-Emby-Token", apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return &TestResultOutput{Body: TestResult{Ok: "false", Message: err.Error()}}, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized {
			return &TestResultOutput{Body: TestResult{Ok: "false", Message: "invalid API key"}}, nil
		}
		if resp.StatusCode != http.StatusOK {
			return &TestResultOutput{Body: TestResult{Ok: "false", Message: fmt.Sprintf("unexpected status %d", resp.StatusCode)}}, nil
		}
		return &TestResultOutput{Body: TestResult{Ok: "true", Message: "connected"}}, nil
	}
}

func GetSettingsSchema() func(context.Context, *struct{}) (*SettingsSchemaOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*SettingsSchemaOutput, error) {
		visible := make([]config.SettingDef, 0, len(config.Schema))
		for _, def := range config.Schema {
			if !def.Hidden {
				visible = append(visible, def)
			}
		}
		return &SettingsSchemaOutput{Body: visible}, nil
	}
}

func GetSettings() func(context.Context, *struct{}) (*SettingsOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*SettingsOutput, error) {
		settings, err := config.GetSettingsMap(db.DB)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to retrieve settings", err)
		}
		delete(settings, "authPasswordHash")
		delete(settings, "authUsername")
		return &SettingsOutput{Body: settings}, nil
	}
}

func UpdateSettings() func(context.Context, *UpdateSettingsInput) (*struct{}, error) {
	return func(ctx context.Context, input *UpdateSettingsInput) (*struct{}, error) {
		// Validate everything first so one bad value can't leave a partial write.
		// Unknown keys (not in the schema) are ignored rather than persisted.
		toWrite := make(map[string]string, len(input.Body))
		for key, value := range input.Body {
			known, err := config.ValidateSetting(key, value)
			if err != nil {
				return nil, huma.Error400BadRequest(err.Error())
			}
			if known {
				toWrite[key] = value
			}
		}
		for key, value := range toWrite {
			if err := config.SetSetting(db.DB, key, value); err != nil {
				return nil, huma.Error500InternalServerError("failed to update settings", err)
			}
		}
		return nil, nil
	}
}
