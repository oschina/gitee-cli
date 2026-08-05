package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gitee.com/oschina/gitee-cli/internal/build"
	"gitee.com/oschina/gitee-cli/internal/config"
)

type swaggerDoc struct {
	Paths map[string]map[string]swaggerOp `json:"paths"`
}

type swaggerOp struct {
	Tags       []string       `json:"tags"`
	Summary    string         `json:"summary"`
	Parameters []swaggerParam `json:"parameters"`
}

type swaggerParam struct {
	Name        string        `json:"name"`
	In          string        `json:"in"`
	Type        string        `json:"type"`
	Format      string        `json:"format"`
	Required    bool          `json:"required"`
	Description string        `json:"description"`
	Enum        []interface{} `json:"enum"`
	Items       *swaggerItems `json:"items"`
}

type swaggerItems struct {
	Type string `json:"type"`
}

type endpointMatch struct {
	Method  string
	Path    string
	Summary string
	Tags    string
	Params  []swaggerParam
}

func fetchSwaggerDoc(client *http.Client) (*swaggerDoc, error) {
	req, err := http.NewRequest(http.MethodGet, config.APISwaggerURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", build.UserAgent())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch swagger doc: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("fetch swagger doc: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read swagger doc: %w", err)
	}

	var doc swaggerDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse swagger doc: %w", err)
	}
	return &doc, nil
}

func searchEndpoints(doc *swaggerDoc, keyword string, limit int) []endpointMatch {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil
	}

	// Split into individual words and require all to match
	words := strings.Fields(strings.ToLower(keyword))
	if len(words) == 0 {
		return nil
	}

	var matches []endpointMatch
	for rawPath, methods := range doc.Paths {
		for method, op := range methods {
			haystack := strings.ToLower(rawPath + " " + method + " " + op.Summary + " " + strings.Join(op.Tags, " "))
			allMatch := true
			for _, w := range words {
				if !strings.Contains(haystack, w) {
					allMatch = false
					break
				}
			}
			if allMatch {
				matches = append(matches, endpointMatch{
					Method:  strings.ToUpper(method),
					Path:    strings.TrimPrefix(rawPath, "/v5"),
					Summary: op.Summary,
					Tags:    strings.Join(op.Tags, ", "),
					Params:  op.Parameters,
				})
				if limit > 0 && len(matches) >= limit {
					return matches
				}
			}
		}
	}
	// limit <= 0 means return all matches
	return matches
}

func printEndpoints(out io.Writer, matches []endpointMatch) {
	if len(matches) == 0 {
		fmt.Fprintln(out, "No matching endpoints found. Try a more general keyword.")
		return
	}
	fmt.Fprintf(out, "Found %d matching endpoint(s):\n\n", len(matches))
	for _, m := range matches {
		fmt.Fprintf(out, "  %-6s %s\n", m.Method, m.Path)
		if m.Summary != "" {
			fmt.Fprintf(out, "        %s\n", m.Summary)
		}
		if m.Tags != "" {
			fmt.Fprintf(out, "        [%s]\n", m.Tags)
		}
		for _, p := range m.Params {
			// Skip access_token (injected automatically)
			if p.Name == "access_token" {
				continue
			}
			reqFlag := " "
			if p.Required {
				reqFlag = "*"
			}
			typed := p.Type
			if p.Items != nil {
				typed = "array[" + p.Items.Type + "]"
			}
			info := fmt.Sprintf("        %s %s (%s, %s)", reqFlag, p.Name, typed, loc(p.In))
			if len(p.Enum) > 0 {
				enumStrs := make([]string, len(p.Enum))
				for i, e := range p.Enum {
					enumStrs[i] = fmt.Sprintf("%v", e)
				}
				info += " one of: " + strings.Join(enumStrs, "|")
			}
			if p.Description != "" {
				info += " - " + p.Description
			}
			fmt.Fprintln(out, info)
		}
		fmt.Fprintln(out)
	}
}

func loc(in string) string {
	switch in {
	case "path":
		return "path"
	case "query":
		return "query"
	case "formData":
		return "form"
	case "body":
		return "body"
	default:
		return in
	}
}

func paginate(all []endpointMatch, page, limit int) ([]endpointMatch, int) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	start := (page - 1) * limit
	if start > len(all) {
		start = len(all)
	}
	end := start + limit
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], len(all)
}

func printPagination(out io.Writer, total, page, limit int) {
	if total == 0 {
		return
	}
	totalPages := (total + limit - 1) / limit
	if totalPages == 1 {
		return
	}
	fmt.Fprintf(out, "\nPage %d/%d (total %d endpoints). Use --page to navigate, --limit to change page size.\n",
		page, totalPages, total)
}
