package build

import "testing"

func TestUserAgent(t *testing.T) {
	originalVersion := Version
	t.Cleanup(func() { Version = originalVersion })

	tests := []struct {
		version string
		want    string
	}{
		{version: "v1.2.3", want: "gitee-cli@1.2.3"},
		{version: "1.2.3-rc.1", want: "gitee-cli@1.2.3-rc.1"},
		{version: "dev+abc1234", want: "gitee-cli@dev+abc1234"},
		{version: "", want: "gitee-cli@dev"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			Version = tt.version
			if got := UserAgent(); got != tt.want {
				t.Fatalf("UserAgent() = %q, want %q", got, tt.want)
			}
		})
	}
}
