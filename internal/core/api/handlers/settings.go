package handlers

import (
	"context"
	"encoding/json"
	"net/http"

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
		resp, err := http.Post(indexerAddr+"/test", "application/json", nil)
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
		for key, value := range input.Body {
			if err := config.SetSetting(db.DB, key, value); err != nil {
				return nil, huma.Error500InternalServerError("failed to update settings", err)
			}
		}
		return nil, nil
	}
}
