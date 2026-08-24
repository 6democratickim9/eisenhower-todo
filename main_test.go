package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestCleanTags(t *testing.T) {
	got := cleanTags([]string{" 업무 ", "개발", "업무", "", "개발 "})
	want := []string{"업무", "개발"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cleanTags() = %#v, want %#v", got, want)
	}
}

func TestFlattenCommentPreservesParagraphBreaks(t *testing.T) {
	adf := map[string]any{
		"type": "doc",
		"content": []any{
			map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": "첫 줄"}}},
			map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": "둘째 줄"}}},
		},
	}
	got := flattenComment(adf)
	if !strings.Contains(got, "첫 줄\n둘째 줄") {
		t.Fatalf("flattenComment() = %q", got)
	}
}
