package client

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RepoResult represents a repo's architecture documentation.
type RepoResult struct {
	Name       string           `json:"name"`
	Sections   []SectionResult  `json:"sections"`
	SourcePath string           `json:"sourcePath,omitempty"`
}

// SectionResult represents a section within a repo's results.
type SectionResult struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// ResultsReader reads architecture results from the arch-hub directory.
type ResultsReader struct {
	ArchHubPath string
}

// NewResultsReader creates a reader for a local arch-hub.
func NewResultsReader(path string) *ResultsReader {
	return &ResultsReader{ArchHubPath: path}
}

// ListRepos returns all repos with .arch.md files.
func (r *ResultsReader) ListRepos() ([]RepoResult, error) {
	entries, err := os.ReadDir(r.ArchHubPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read arch-hub at %s: %w", r.ArchHubPath, err)
	}

	var repos []RepoResult
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		archFiles, _ := filepath.Glob(filepath.Join(r.ArchHubPath, e.Name(), "*.arch.md"))
		if len(archFiles) > 0 {
			repo := RepoResult{
				Name:       e.Name(),
				SourcePath: filepath.Join(r.ArchHubPath, e.Name()),
			}
			for _, f := range archFiles {
				data, err := os.ReadFile(f)
				if err != nil {
					continue
				}
				sections := parseSections(string(data))
				repo.Sections = append(repo.Sections, sections...)
			}
			repos = append(repos, repo)
		}
	}

	sort.Slice(repos, func(i, j int) bool { return repos[i].Name < repos[j].Name })
	return repos, nil
}

// ReadRepo returns the full content for a specific repo.
func (r *ResultsReader) ReadRepo(name string) (*RepoResult, error) {
	repoDir := filepath.Join(r.ArchHubPath, name)
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("repo %q not found in arch-hub", name)
	}

	archFiles, _ := filepath.Glob(filepath.Join(repoDir, "*.arch.md"))
	if len(archFiles) == 0 {
		return nil, fmt.Errorf("no .arch.md files for repo %q", name)
	}

	repo := &RepoResult{
		Name:       name,
		SourcePath: repoDir,
	}
	for _, f := range archFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		sections := parseSections(string(data))
		repo.Sections = append(repo.Sections, sections...)
	}
	return repo, nil
}

// SearchRepos searches across all repos for a query string.
func (r *ResultsReader) SearchRepos(query string) ([]SearchHit, error) {
	repos, err := r.ListRepos()
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	var hits []SearchHit

	for _, repo := range repos {
		for _, sec := range repo.Sections {
			lines := strings.Split(sec.Content, "\n")
			for i, line := range lines {
				if strings.Contains(strings.ToLower(line), queryLower) {
					hits = append(hits, SearchHit{
						Repo:    repo.Name,
						Section: sec.Name,
						Line:    i + 1,
						Text:    strings.TrimSpace(line),
					})
				}
			}
		}
	}
	return hits, nil
}

// SearchHit represents a search match.
type SearchHit struct {
	Repo    string `json:"repo"`
	Section string `json:"section"`
	Line    int    `json:"line"`
	Text    string `json:"text"`
}

// ExportRepo exports a repo's results as a single markdown file.
func (r *ResultsReader) ExportRepo(name string) (string, error) {
	repo, err := r.ReadRepo(name)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s Architecture\n\n", name))
	for _, sec := range repo.Sections {
		sb.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", sec.Name, sec.Content))
	}
	return sb.String(), nil
}

// parseSections splits an .arch.md file into sections by top-level headings.
func parseSections(content string) []SectionResult {
	lines := strings.Split(content, "\n")
	var sections []SectionResult
	var current *SectionResult
	var buf []string

	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			// Save previous section
			if current != nil {
				current.Content = strings.TrimSpace(strings.Join(buf, "\n"))
				sections = append(sections, *current)
			}
			current = &SectionResult{Name: strings.TrimPrefix(line, "# ")}
			buf = nil
		} else if current != nil {
			buf = append(buf, line)
		} else {
			// Content before first heading — treat as "overview"
			if strings.TrimSpace(line) != "" {
				if current == nil {
					current = &SectionResult{Name: "overview"}
				}
				buf = append(buf, line)
			}
		}
	}

	if current != nil {
		current.Content = strings.TrimSpace(strings.Join(buf, "\n"))
		sections = append(sections, *current)
	}

	return sections
}
