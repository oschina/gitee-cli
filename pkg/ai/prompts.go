package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type PRDraft struct {
	Title string
	Body  string
}

func GeneratePRDraft(ctx context.Context, client *Client, diff, commits, language, template string) (*PRDraft, error) {
	lang := language
	if lang == "" {
		lang = "English"
	}

	system := fmt.Sprintf(`You are an expert software engineer helping write pull request descriptions.
Analyze the provided git diff and commit log, then generate a concise PR title and body.

Rules:
- Title: one line, imperative mood, max 72 chars, no period at end
- Body: markdown format, explain WHAT changed and WHY, include a brief summary and key changes
- Language: write entirely in %s
- Be specific and factual, avoid vague phrases like "various improvements"
- Output ONLY valid JSON in this exact format, nothing else:
{"title": "...", "body": "..."}`, lang)

	if template != "" {
		system += fmt.Sprintf(`

IMPORTANT: You must fill in the following PR template for the body field. Keep all
section headers, checkbox items, and structure exactly as-is. Only replace placeholder
text with actual content derived from the diff and commits. The "body" in your JSON
output must follow this template structure:

%s`, template)
	}

	var sb strings.Builder
	if commits != "" {
		sb.WriteString("## Commit log\n")
		sb.WriteString(commits)
		sb.WriteString("\n\n")
	}
	sb.WriteString("## Git diff\n```diff\n")
	sb.WriteString(diff)
	sb.WriteString("\n```")

	raw, err := client.Complete(ctx, system, sb.String())
	if err != nil {
		return nil, err
	}

	return parsePRDraft(raw)
}

func parsePRDraft(raw string) (*PRDraft, error) {
	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("ai: could not find JSON in response: %q", raw)
	}
	raw = raw[start : end+1]

	var result struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("ai: parse draft JSON: %w", err)
	}
	if result.Title == "" {
		return nil, fmt.Errorf("ai: generated title is empty")
	}
	return &PRDraft{
		Title: strings.TrimSpace(result.Title),
		Body:  strings.TrimSpace(result.Body),
	}, nil
}

func GeneratePRReview(ctx context.Context, client *Client, prTitle, prBody, diff, language string) (string, error) {
	lang := language
	if lang == "" {
		lang = "English"
	}

	system := fmt.Sprintf(`You are an experienced software engineer performing a thorough code review.
Analyze the pull request and provide structured, actionable feedback.

Output format (Markdown):
## Summary
One paragraph summarizing what this PR does.

## Review

### 🔴 Issues (must fix before merge)
List blocking problems: bugs, security issues, incorrect logic. If none, write "None."

### 🟡 Suggestions (non-blocking improvements)
List style, performance, readability suggestions. If none, write "None."

### 🟢 Positives
Note good patterns or well-written code worth acknowledging.

## Verdict
One of: **Approve** / **Request Changes** / **Needs Discussion**

Rules:
- Be specific: reference file names or code patterns when possible
- Language: write entirely in %s
- Do not repeat the diff verbatim`, lang)

	var sb strings.Builder
	if prTitle != "" {
		sb.WriteString("## PR Title\n")
		sb.WriteString(prTitle)
		sb.WriteString("\n\n")
	}
	if prBody != "" {
		sb.WriteString("## PR Description\n")
		sb.WriteString(prBody)
		sb.WriteString("\n\n")
	}
	sb.WriteString("## Diff\n```diff\n")
	sb.WriteString(diff)
	sb.WriteString("\n```")

	return client.Complete(ctx, system, sb.String())
}

func GenerateIssueDraft(ctx context.Context, client *Client, description, language string) (*IssueDraft, error) {
	lang := language
	if lang == "" {
		lang = "English"
	}

	system := fmt.Sprintf(`You are a helpful assistant that writes structured software issue reports.
Expand the user's brief description into a well-formatted issue.

Rules:
- Title: concise, specific, max 72 chars
- Body: markdown, include relevant sections from: Summary, Steps to Reproduce, Expected Behavior, Actual Behavior, Environment (if applicable)
- Only include sections that are relevant to the description
- Language: write entirely in %s
- Output ONLY valid JSON, nothing else:
{"title": "...", "body": "..."}`, lang)

	raw, err := client.Complete(ctx, system, description)
	if err != nil {
		return nil, err
	}

	return parseIssueDraft(raw)
}

type IssueDraft struct {
	Title string
	Body  string
}

func parseIssueDraft(raw string) (*IssueDraft, error) {
	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("ai: could not find JSON in response: %q", raw)
	}
	raw = raw[start : end+1]

	var result struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("ai: parse issue draft JSON: %w", err)
	}
	if result.Title == "" {
		return nil, fmt.Errorf("ai: generated title is empty")
	}
	return &IssueDraft{
		Title: strings.TrimSpace(result.Title),
		Body:  strings.TrimSpace(result.Body),
	}, nil
}
