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

// Mode distinguishes checkout links from portable package references.
type Mode string

const (
	// Local renders manifests while supporting resources remain linked to source.
	Local Mode = "local"
	// Native renders every Markdown file for a self-contained package.
	Native Mode = "native"
)

// Options supplies explicit destinations and selections for a text transform.
type Options struct {
	Mode                                      Mode
	Provider, Package, SkillsRoot, OutputRoot string
	Selected                                  []string
}

var links = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
var scheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)
var invocations = regexp.MustCompile(`\$[a-z0-9]+(-[a-z0-9]+)*`)

// Markdown renders one source-relative Markdown file without writing files.
// Local supporting Markdown is returned unchanged because it remains source-linked.
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
	if opts.Mode == Local && relative != "SKILL.md" {
		return text, nil
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
		if opts.Mode == Local && (!strings.HasPrefix(reference, "../") || !strings.HasSuffix(reference, "/SKILL.md")) {
			return link
		}
		resolved := filepath.Clean(filepath.Join(filepath.Dir(source), strings.SplitN(reference, "#", 2)[0]))
		for _, target := range c.Skills {
			if resolved != filepath.Join(target.Source, "SKILL.md") {
				continue
			}
			if opts.Mode == Native && !selected[target.Name] {
				return "`" + target.Name + "` (optional integration)"
			}
			path := filepath.Join(target.Source, "SKILL.md")
			if selected[target.Name] {
				path = filepath.Join(opts.SkillsRoot, target.Name, "SKILL.md")
			}
			if opts.Mode == Local {
				return "[" + parts[1] + "](<" + path + ">)"
			}
			rel, relErr := filepath.Rel(filepath.Dir(output), path)
			if relErr != nil {
				return link
			}
			return "[" + parts[1] + "](" + filepath.ToSlash(rel) + ")"
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
	if opts.Mode == Native {
		original := ".agents/skills/coding/python-standards/scripts/check_function_size.py"
		if strings.Contains(text, original) {
			checker, err := filepath.Rel(filepath.Dir(output), filepath.Join(opts.SkillsRoot, name, "scripts/check_function_size.py"))
			if err != nil {
				return "", err
			}
			text = strings.ReplaceAll(text, original, `"<loaded-reference-directory>/`+filepath.ToSlash(checker)+`"`)
			text += "\nResolve `<loaded-reference-directory>` to the directory containing this loaded reference before running the bundled checker.\n"
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
	policy, err := os.ReadFile(filepath.Join(c.Root, ".agents/global/instructions.md"))
	if err != nil {
		return "", err
	}
	_, tail, ok := strings.Cut(string(policy), "## Markdown artifact outputs")
	if !ok {
		return "", fmt.Errorf("missing Markdown artifact output policy")
	}
	text += "\n## Markdown artifact outputs" + tail
	if opts.Mode == Native {
		text += "\n### Native output policy\n\nHonor explicit user guidance configuring an output location, including a centralized output root. Otherwise use `<repo-root>/<branch-slug>/<skill-name>/...`. These task-local defaults do not modify global settings.\n"
	} else {
		text += "\n### Output location for this skill\n\n"
		if opts.OutputRoot != "" {
			text += "The configured centralized output root is `" + opts.OutputRoot + "`.\nUse `" + opts.OutputRoot + "/<repo-slug>/<branch-slug>/<skill-name>/...` unless the user supplies a path.\n"
		} else {
			text += "Use `<repo-root>/<branch-slug>/<skill-name>/...` unless the user supplies a path.\nNo centralized output root is configured for this installation.\n"
		}
		text += "These rules apply only while performing this skill.\n"
	}
	return text, nil
}
