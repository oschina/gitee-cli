package skillassets

import (
	"io/fs"
	"reflect"
	"testing"
)

func TestNames(t *testing.T) {
	want := []string{"gitee-api", "gitee-issue", "gitee-pr", "gitee-release", "gitee-repo", "gitee-search"}
	got, err := Names()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("embedded skills = %v, want %v", got, want)
	}
	for _, name := range got {
		data, err := fs.ReadFile(Files, name+"/SKILL.md")
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if len(data) == 0 {
			t.Fatalf("%s/SKILL.md is empty", name)
		}
	}
}
