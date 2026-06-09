package handlers

import "context"

func GetVersion(version string) func(context.Context, *struct{}) (*VersionOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*VersionOutput, error) {
		return &VersionOutput{Body: VersionResponse{Version: version}}, nil
	}
}
