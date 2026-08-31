package i18n

import (
	"fmt"
	"testing"
)

// The pipeline.commit.confirm call site (pkg/cmd/pipeline/commit.go) passes
// (fileName, ref, owner/repo); the zh word order puts repo before branch, so
// the template must consume the args via positional verbs.
func TestZhCommitConfirmArgOrder(t *testing.T) {
	got := fmt.Sprintf(zhCN["pipeline.commit.confirm"], "ci.yml", "master", "owner/repo")
	want := "将流水线 YAML \"ci.yml\" 提交到 owner/repo 的 \"master\" 分支？"
	if got != want {
		t.Errorf("got %s", got)
	}
}
