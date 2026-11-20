// Package render transforms skill Markdown for local installations and portable
// native packages. Callers own filesystem materialization and resource safety.
package render

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/EimantasZilis/agent-toolkit/internal/catalog"
)

// Mode distinguishes local installations from portable package references.
type Mode string

const (
	// Local renders Markdown for installed skills, linking to checkout settings.
	Local Mode = "local"
	// Native renders every Markdown file for a self-contained package.
	Native Mode = "native"
)

// Options supplies explicit destinations and selections for a text transform.
// SourceRoot is the persistent checkout for local settings links; when empty,
// it defaults to the catalog root. Native output always uses package-local links.
type Options struct {
	Mode                                      Mode
	Provider, Package, SkillsRoot, SourceRoot string
	Selected                                  []string
}

var links = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
var scheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)
var invocations = regexp.MustCompile(`\$[a-z0-9]+(-[a-z0-9]+)*`)

// Markdown renders one source-relative Markdown file without writing files.
func Markdown(c *catalog.Catalog, name, relative, text string, opts Options) (string, error) {
	if opts.Mode != Local && opts.Mode != Native {
		return "", fmt.Errorf("unknown rendering mode: %s", opts.Mode)
	}
	if opts.Provider != "codex" && opts.Provider != "claude" {
		return "", fmt.Errorf("unknown provider: %s", opts.Provider)
	}
	skill, err := c.Lookup(name)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(filepath.Clean(relative), ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid source-relative path: %s", relative)
	}
	selected := map[string]bool{}
	for _, item := range opts.Selected {
		if _, err := c.Lookup(item); err != nil {
			return "", err
		}
		selected[item] = true
	}
	source := filepath.Join(skill.Source, relative)
	output := filepath.Join(opts.SkillsRoot, name, relative)
	text = links.ReplaceAllStringFunc(text, func(link string) string {
		parts := links.FindStringSubmatch(link)
		reference := strings.Trim(parts[2], "<>")
		if scheme.MatchString(reference) || strings.HasPrefix(reference, "#") {
			return link
		}
		referencePath, fragment, hasFragment := strings.Cut(reference, "#")
		suffix := ""
		if hasFragment {
			suffix = "#" + fragment
		}
		resolved := filepath.Clean(filepath.Join(filepath.Dir(source), referencePath))
		for _, target := range c.Skills {
			rel, err := filepath.Rel(target.Source, resolved)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				continue
			}
			if !selected[target.Name] && target.Name != name {
				return "`" + target.Name + "` (optional integration)"
			}
			destination := filepath.Join(opts.SkillsRoot, target.Name, rel)
			if opts.Mode == Local {
				return "[" + parts[1] + "](<" + destination + suffix + ">)"
			}
			local, err := filepath.Rel(filepath.Dir(output), destination)
			if err != nil {
				return link
			}
			result := filepath.ToSlash(local) + suffix
			if strings.Contains(result, " ") {
				result = "<" + result + ">"
			}
			return "[" + parts[1] + "](" + result + ")"
		}
		return link
	})
	if opts.Provider == "claude" {
		text = invocations.ReplaceAllStringFunc(text, func(token string) string {
			name := token[1:]
			if _, err := c.Lookup(name); err != nil {
				return token
			}
			if opts.Mode == Local {
				return "/" + name
			}
			if selected[name] {
				return "/" + opts.Package + ":" + name
			}
			return "`" + name + "`"
		})
	}
	{
		original := ".agents/skills/coding/python-standards/scripts/check_function_size.py"
		if strings.Contains(text, original) {
			checker, err := filepath.Rel(filepath.Dir(output), filepath.Join(opts.SkillsRoot, name, "scripts/check_function_size.py"))
			if err != nil {
				return "", err
			}
			if opts.Mode == Local {
				installed := filepath.Join(opts.SkillsRoot, name, "scripts/check_function_size.py")
				text = strings.ReplaceAll(text, original, "'"+strings.ReplaceAll(installed, "'", "'\"'\"'")+"'")
			} else {
				text = strings.ReplaceAll(text, original, `"<loaded-reference-directory>/`+filepath.ToSlash(checker)+`"`)
				text += "\nResolve `<loaded-reference-directory>` to the directory containing this loaded reference before running the bundled checker.\n"
			}
		}
	}
	if relative != "SKILL.md" {
		return text, nil
	}
	if opts.Provider == "claude" && skill.Explicit && !strings.Contains(text, "\ndisable-model-invocation:") {
		text = strings.Replace(text, "\n---", "\ndisable-model-invocation: true\n---", 1)
	}
	if !skill.Output {
		return text, nil
	}
	if _, err := os.ReadFile(filepath.Join(c.Root, ".agents/global/markdown-output.md")); err != nil {
		return "", fmt.Errorf("load Markdown output instructions: %w", err)
	}
	text += "\n## Markdown output settings\n\n"
	if opts.Mode == Native {
		text += "Before writing or finding Markdown artifacts, load and apply the [Markdown output instructions](../../settings/markdown-output.md).\n"
	} else {
		root := opts.SourceRoot
		if root == "" {
			root = c.Root
		}
		path := filepath.ToSlash(filepath.Join(root, ".agents/global/markdown-output.md"))
		text += "Before writing or finding Markdown artifacts, load and apply the [Markdown output instructions](<" + path + ">).\nUse the artifact location configured in global AGENTS.md or CLAUDE.md.\n"
	}
	return text, nil
}
