package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/reposwarm/ask/internal/client"
	"github.com/reposwarm/ask/internal/config"
	"github.com/reposwarm/ask/internal/output"
)

var resultsCmd = &cobra.Command{
	Use:     "results",
	Aliases: []string{"res"},
	Short:   "Browse architecture investigation results",
	Long: `Browse .arch.md files from the arch-hub.

Reads directly from the local arch-hub directory. If askbox is running,
uses its arch-hub path. Otherwise, specify --path.

Examples:
  ask results list
  ask results read my-api
  ask results read my-api overview
  ask results search "DynamoDB"
  ask results export my-api -o out.md
  ask results export --all -d ./docs`,
}

var (
	resultsPathFlag string
)

func init() {
	resultsCmd.PersistentFlags().StringVar(&resultsPathFlag, "path", "", "Arch-hub directory path (default: auto-detect from askbox)")

	resultsCmd.AddCommand(resultsListCmd)
	resultsCmd.AddCommand(resultsReadCmd)
	resultsCmd.AddCommand(resultsSearchCmd)
	resultsCmd.AddCommand(resultsExportCmd)
	resultsCmd.AddCommand(resultsDiffCmd)

	rootCmd.AddCommand(resultsCmd)
}

// getResultsReader resolves the arch-hub path and returns a reader.
func getResultsReader(cmd *cobra.Command) (*client.ResultsReader, error) {
	// Explicit path flag
	if resultsPathFlag != "" {
		return client.NewResultsReader(resultsPathFlag), nil
	}

	// Try askbox health to get arch-hub path
	serverURL := getServerURL(cmd)
	c := client.New(serverURL)
	health, err := c.Health()
	if err == nil && health.ArchHubReady {
		// Askbox is running — but we need local access to the files
		// If askbox is local (Docker volume), the path isn't directly accessible
		// Fall back to common locations
	}

	// Check common locations
	paths := []string{
		"/tmp/arch-hub",                                       // Docker volume default
		config.DataDir() + "/arch-hub",                        // ask data dir
	}

	// Check RepoSwarm install
	home, _ := os.UserHomeDir()
	paths = append(paths, home+"/.reposwarm/arch-hub")
	paths = append(paths, home+"/reposwarm/arch-hub")

	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			files, _ := os.ReadDir(p)
			if len(files) > 0 {
				return client.NewResultsReader(p), nil
			}
		}
	}

	// If askbox is running but we can't find files locally,
	// use the askbox API to get results
	if err == nil && health.ArchHubReady {
		return nil, fmt.Errorf("askbox has arch-hub loaded (%d repos) but files not accessible locally\n  Use: ask results --path /path/to/arch-hub", health.ArchHubRepos)
	}

	return nil, fmt.Errorf("no arch-hub found\n  Use: ask results --path /path/to/arch-hub\n  Or:  ask refresh --url <git-url>  to load one into askbox")
}

// ── results list ──

var resultsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List repos with architecture docs",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader, err := getResultsReader(cmd)
		if err != nil {
			output.Error(err.Error(), "")
			return err
		}

		repos, err := reader.ListRepos()
		if err != nil {
			return err
		}

		if output.JSONMode {
			return output.JSON(repos)
		}

		if len(repos) == 0 {
			fmt.Println("No repos found in arch-hub.")
			return nil
		}

		fmt.Printf("📚 Architecture Results (%d repos)\n\n", len(repos))
		for _, r := range repos {
			fmt.Printf("  📁 %-30s %d sections\n", r.Name, len(r.Sections))
		}
		return nil
	},
}

// ── results read ──

var resultsReadRawFlag bool

var resultsReadCmd = &cobra.Command{
	Use:   "read <repo> [section]",
	Short: "Read architecture docs for a repo",
	Long: `Read .arch.md content for a repository.

Without a section: shows all sections.
With a section name: shows just that section.

Examples:
  ask results read my-api
  ask results read my-api overview
  ask results read my-api --raw > out.md`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		reader, err := getResultsReader(cmd)
		if err != nil {
			return err
		}

		repo, err := reader.ReadRepo(args[0])
		if err != nil {
			return err
		}

		// Filter to specific section
		if len(args) == 2 {
			sectionName := args[1]
			for _, s := range repo.Sections {
				if strings.EqualFold(s.Name, sectionName) {
					if output.JSONMode {
						return output.JSON(s)
					}
					if resultsReadRawFlag {
						fmt.Print(s.Content)
						return nil
					}
					fmt.Printf("📄 %s / %s\n\n%s\n", repo.Name, s.Name, s.Content)
					return nil
				}
			}
			return fmt.Errorf("section %q not found in %s", sectionName, repo.Name)
		}

		// All sections
		if output.JSONMode {
			return output.JSON(repo)
		}

		if resultsReadRawFlag {
			for _, s := range repo.Sections {
				fmt.Printf("# %s\n\n%s\n\n", s.Name, s.Content)
			}
			return nil
		}

		fmt.Printf("📄 %s (%d sections)\n\n", repo.Name, len(repo.Sections))
		for _, s := range repo.Sections {
			fmt.Printf("── %s ──\n\n%s\n\n", s.Name, s.Content)
		}
		return nil
	},
}

// ── results search ──

var (
	searchRepoFlag string
	searchMaxFlag  int
)

var resultsSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search across architecture docs",
	Long: `Search for text across all .arch.md files.

Examples:
  ask results search "DynamoDB"
  ask results search "auth" --repo my-api
  ask results search "security" --max 20`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reader, err := getResultsReader(cmd)
		if err != nil {
			return err
		}

		hits, err := reader.SearchRepos(args[0])
		if err != nil {
			return err
		}

		// Filter by repo
		if searchRepoFlag != "" {
			var filtered []client.SearchHit
			for _, h := range hits {
				if h.Repo == searchRepoFlag {
					filtered = append(filtered, h)
				}
			}
			hits = filtered
		}

		// Limit
		if searchMaxFlag > 0 && len(hits) > searchMaxFlag {
			hits = hits[:searchMaxFlag]
		}

		if output.JSONMode {
			return output.JSON(hits)
		}

		if len(hits) == 0 {
			fmt.Printf("No results for %q\n", args[0])
			return nil
		}

		fmt.Printf("🔍 %d matches for %q\n\n", len(hits), args[0])
		for _, h := range hits {
			text := h.Text
			if len(text) > 100 {
				text = text[:97] + "..."
			}
			fmt.Printf("  %s/%s:%d  %s\n", h.Repo, h.Section, h.Line, text)
		}
		return nil
	},
}

// ── results export ──

var (
	exportOutputFlag string
	exportDirFlag    string
	exportAllFlag    bool
)

var resultsExportCmd = &cobra.Command{
	Use:   "export [repo]",
	Short: "Export architecture docs as markdown",
	Long: `Export .arch.md content to local files.

Single repo:
  ask results export my-api              # stdout
  ask results export my-api -o out.md    # file

All repos:
  ask results export --all -d ./docs     # directory`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reader, err := getResultsReader(cmd)
		if err != nil {
			return err
		}

		if exportAllFlag {
			dir := exportDirFlag
			if dir == "" {
				dir = "."
			}
			repos, err := reader.ListRepos()
			if err != nil {
				return err
			}
			os.MkdirAll(dir, 0755)
			exported := 0
			for _, r := range repos {
				md, err := reader.ExportRepo(r.Name)
				if err != nil {
					output.Error(fmt.Sprintf("Failed to export %s: %s", r.Name, err), "")
					continue
				}
				dest := fmt.Sprintf("%s/%s.arch.md", dir, r.Name)
				if err := os.WriteFile(dest, []byte(md), 0644); err != nil {
					output.Error(fmt.Sprintf("Failed to write %s: %s", dest, err), "")
					continue
				}
				output.Success(fmt.Sprintf("%s (%d bytes)", r.Name, len(md)))
				exported++
			}
			output.Success(fmt.Sprintf("Exported %d/%d repos to %s", exported, len(repos), dir))
			return nil
		}

		if len(args) == 0 {
			return fmt.Errorf("provide a repo name or use --all")
		}

		md, err := reader.ExportRepo(args[0])
		if err != nil {
			return err
		}

		dest := exportOutputFlag
		if dest == "" && exportDirFlag != "" {
			dest = fmt.Sprintf("%s/%s.arch.md", exportDirFlag, args[0])
		}

		if dest != "" {
			if exportDirFlag != "" {
				os.MkdirAll(exportDirFlag, 0755)
			}
			if err := os.WriteFile(dest, []byte(md), 0644); err != nil {
				return err
			}
			output.Success(fmt.Sprintf("Exported to %s (%d bytes)", dest, len(md)))
			return nil
		}

		fmt.Print(md)
		return nil
	},
}

// ── results diff ──

var resultsDiffCmd = &cobra.Command{
	Use:   "diff <repo1> <repo2>",
	Short: "Compare architecture docs between two repos",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		reader, err := getResultsReader(cmd)
		if err != nil {
			return err
		}

		r1, err := reader.ReadRepo(args[0])
		if err != nil {
			return err
		}
		r2, err := reader.ReadRepo(args[1])
		if err != nil {
			return err
		}

		set1 := map[string]bool{}
		for _, s := range r1.Sections {
			set1[s.Name] = true
		}
		set2 := map[string]bool{}
		for _, s := range r2.Sections {
			set2[s.Name] = true
		}

		if output.JSONMode {
			var only1, only2, shared []string
			for k := range set1 {
				if set2[k] {
					shared = append(shared, k)
				} else {
					only1 = append(only1, k)
				}
			}
			for k := range set2 {
				if !set1[k] {
					only2 = append(only2, k)
				}
			}
			return output.JSON(map[string]any{
				"repo1": args[0], "repo2": args[1],
				"sections1": len(r1.Sections), "sections2": len(r2.Sections),
				"onlyIn1": only1, "onlyIn2": only2, "shared": shared,
			})
		}

		fmt.Printf("📊 Diff: %s vs %s\n\n", args[0], args[1])
		fmt.Printf("  %-30s %d sections\n", args[0], len(r1.Sections))
		fmt.Printf("  %-30s %d sections\n\n", args[1], len(r2.Sections))

		allSections := map[string]bool{}
		for k := range set1 {
			allSections[k] = true
		}
		for k := range set2 {
			allSections[k] = true
		}

		for s := range allSections {
			in1 := set1[s]
			in2 := set2[s]
			switch {
			case in1 && in2:
				fmt.Printf("  ✅ %-30s  both\n", s)
			case in1:
				fmt.Printf("  ◀️  %-30s  only in %s\n", s, args[0])
			case in2:
				fmt.Printf("  ▶️  %-30s  only in %s\n", s, args[1])
			}
		}
		return nil
	},
}

func init() {
	resultsReadCmd.Flags().BoolVar(&resultsReadRawFlag, "raw", false, "Raw markdown output")

	resultsSearchCmd.Flags().StringVar(&searchRepoFlag, "repo", "", "Filter by repo")
	resultsSearchCmd.Flags().IntVar(&searchMaxFlag, "max", 50, "Maximum results")

	resultsExportCmd.Flags().StringVarP(&exportOutputFlag, "output", "o", "", "Output file")
	resultsExportCmd.Flags().StringVarP(&exportDirFlag, "dir", "d", "", "Output directory")
	resultsExportCmd.Flags().BoolVar(&exportAllFlag, "all", false, "Export all repos")
}
