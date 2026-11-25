// Command stickyeval measures whether agents use the board right: the
// cases in plugin/evals, graded the same way for Claude Code and Codex,
// with the plugin and without it. It is a tool for this repository, not
// part of stickypane.
//
// With no flags it lists the cases. -plan prints what a run would do and
// what it would cost to know, and runs nothing. Only -run calls a model:
// every run costs money, so the maintainer approves it first.
//
//	go run ./cmd/stickyeval                    # the cases
//	go run ./cmd/stickyeval -plan              # the runs, the models, the caps
//	go run ./cmd/stickyeval -run -agents codex -case 02-steps -k 1 -board DIR
//	go run ./cmd/stickyeval -grade DIR -case 02-steps -board DIR
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

// Case is what this program reads of a case.yaml. The file is written in
// JSON, which YAML reads too: claude plugin eval and this program read
// the same file, with no YAML module.
type Case struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Context     struct {
		Scaffold string `json:"scaffold_script"`
	} `json:"context"`
	Execution struct {
		Prompt  string `json:"prompt"`
		Timeout int    `json:"timeout_seconds"`
	} `json:"execution"`
	Dir string `json:"-"`
}

func loadCases(dir string) ([]Case, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*", "case.yaml"))
	if err != nil {
		return nil, err
	}
	var out []Case
	for _, p := range paths { // Glob sorts: the cases keep their numbers' order
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var c Case
		if err := json.Unmarshal(b, &c); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		c.Dir = filepath.Dir(p)
		out = append(out, c)
	}
	return out, nil
}

// exe runs one stickypane command on the board in dir.
type exe func(dir string, args ...string) error

type options struct {
	evals, only, agents, board, gradeDir, sticky string
	plan, run                                    bool
	k                                            int
	claudeModel, codexModel                      string
	maxUSD                                       float64
	maxTokens                                    int
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, nil)) }

func run(args []string, out io.Writer, x exe) int {
	var o options
	fs := flag.NewFlagSet("stickyeval", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.StringVar(&o.evals, "evals", "plugin/evals", "the folder of the cases")
	fs.StringVar(&o.only, "case", "", "only the cases whose name matches this glob")
	fs.BoolVar(&o.plan, "plan", false, "print what a run would do, and run nothing")
	fs.BoolVar(&o.run, "run", false, "run the cases: calls the models and costs money")
	fs.IntVar(&o.k, "k", 3, "runs of each case in each arm")
	fs.StringVar(&o.agents, "agents", "claude,codex", "the agents to run")
	fs.StringVar(&o.claudeModel, "claude-model", "", "Claude Code's model (default: its own)")
	fs.StringVar(&o.codexModel, "codex-model", "", "Codex's model (default: its config)")
	fs.Float64Var(&o.maxUSD, "max-cost-usd", 5, "Claude Code's cost ceiling, passed to claude plugin eval")
	fs.IntVar(&o.maxTokens, "max-tokens", 2_000_000, "Codex's ceiling: tokens over all its runs, checked before each")
	fs.StringVar(&o.board, "board", "", "a project folder whose board hears the results (never the user's own)")
	fs.StringVar(&o.gradeDir, "grade", "", "grade a recorded run: a folder with events.jsonl, sticky/ and before")
	fs.StringVar(&o.sticky, "stickypane", "stickypane", "the stickypane program the board commands run")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if x == nil {
		x = func(dir string, args ...string) error {
			cmd := exec.Command(o.sticky, args...)
			cmd.Dir = dir
			if b, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("stickypane %s: %v: %s", strings.Join(args, " "), err, b)
			}
			return nil
		}
	}
	cases, err := loadCases(o.evals)
	if err == nil && len(cases) == 0 {
		err = fmt.Errorf("no cases in %s", o.evals)
	}
	if err != nil {
		fmt.Fprintln(out, "stickyeval:", err)
		return 1
	}
	if o.only != "" {
		var keep []Case
		for _, c := range cases {
			if ok, _ := filepath.Match(o.only, c.Name); ok {
				keep = append(keep, c)
			}
		}
		cases = keep
	}
	var results []Result
	switch {
	case o.gradeDir != "":
		if len(cases) != 1 {
			fmt.Fprintln(out, "stickyeval: -grade needs -case naming one case")
			return 2
		}
		f, err := os.Open(filepath.Join(o.gradeDir, "events.jsonl"))
		if err != nil {
			fmt.Fprintln(out, "stickyeval:", err)
			return 1
		}
		events, err := readEvents(f)
		f.Close()
		if err != nil {
			fmt.Fprintln(out, "stickyeval:", err)
			return 1
		}
		r := grade(Run{Case: cases[0].Name, Events: events, Sticky: filepath.Join(o.gradeDir, "sticky"), Before: readBefore(filepath.Join(o.gradeDir, "before"))})
		r.Agent, r.Arm = "recorded", "run"
		results = []Result{r}
	case o.plan || o.run:
		plan(out, cases, o)
		if !o.run {
			return 0
		}
		results, err = runAll(out, cases, o)
		if err != nil {
			fmt.Fprintln(out, "stickyeval:", err)
		}
	default:
		tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "case\tmeasures")
		for _, c := range cases {
			fmt.Fprintf(tw, "%s\t%s\n", c.Name, c.Description)
		}
		tw.Flush()
		return 0
	}
	table(out, results)
	if o.board != "" {
		if err := report(o.board, results, x); err != nil {
			fmt.Fprintln(out, "stickyeval:", err)
			return 1
		}
	}
	return 0
}

func agents(o options) []string { return strings.Split(o.agents, ",") }

// claudeArgs is the one claude plugin eval call that runs every case in
// both arms; it scores them with its own graders and keeps its report.
func claudeArgs(o options) []string {
	args := []string{"plugin", "eval", "plugin", "--runs", fmt.Sprint(o.k), "--ablation", "with-without",
		"--scaffold", "--allow-tools", "Bash", "Write", "Edit", "--max-cost-usd", fmt.Sprint(o.maxUSD), "--no-publish"}
	if o.only != "" {
		args = append(args, "--case", o.only)
	}
	if o.claudeModel != "" {
		args = append(args, "--model", o.claudeModel)
	}
	return args
}

func plan(out io.Writer, cases []Case, o options) {
	n := len(agents(o))
	fmt.Fprintf(out, "plan: %d cases × 2 arms (with, without) × k=%d × %d agents = %d runs\n", len(cases), o.k, n, len(cases)*2*o.k*n)
	model := func(m, def string) string {
		if m == "" {
			return def
		}
		return m
	}
	for _, a := range agents(o) {
		switch a {
		case "claude":
			fmt.Fprintf(out, "claude: claude %s\n", strings.Join(claudeArgs(o), " "))
			fmt.Fprintf(out, "  model: %s; judge: none (no llm graders); cap: $%g\n", model(o.claudeModel, "its default"), o.maxUSD)
		case "codex":
			fmt.Fprintf(out, "codex: per run, in a new repository: codex %s\n", strings.Join(codexArgs("<repo>", "<cwd>", "with", o.codexModel), " "))
			fmt.Fprintln(out, "  the arm without the plugin adds --disable plugins")
			fmt.Fprintf(out, "  model: %s; cap: %d tokens over all its runs, checked before each\n", model(o.codexModel, "its config's"), o.maxTokens)
		default:
			fmt.Fprintf(out, "%s: not an agent this program runs\n", a)
		}
	}
	fmt.Fprintln(out, "a run's cost is not known until the first one: start with -case 02-steps -k 1 -agents codex")
	if !o.run {
		fmt.Fprintln(out, "nothing was run. -run runs this, once the maintainer approves the cost.")
	}
}

func runAll(out io.Writer, cases []Case, o options) ([]Result, error) {
	var results []Result
	for _, a := range agents(o) {
		switch a {
		case "claude":
			// ponytail: Claude's runs are graded by its own graders for now;
			// its trace becomes Events once the first run shows its format.
			cmd := exec.Command("claude", claudeArgs(o)...)
			cmd.Stdout, cmd.Stderr = out, out
			if err := cmd.Run(); err != nil {
				fmt.Fprintln(out, "claude plugin eval:", err)
			}
		case "codex":
			used := 0
			for _, c := range cases {
				for _, arm := range []string{"with", "without"} {
					for i := 0; i < o.k; i++ {
						if used >= o.maxTokens {
							return results, fmt.Errorf("codex stopped at %d tokens (-max-tokens %d)", used, o.maxTokens)
						}
						r, err := runCodex(c, arm, o.codexModel)
						if err != nil {
							return results, fmt.Errorf("%s %s: %w", c.Name, arm, err)
						}
						used += r.Use.Input + r.Use.Output
						fmt.Fprintf(out, "%s codex %s #%d: %v %s\n", c.Name, arm, i+1, r.Pass, r.Reason)
						results = append(results, r)
					}
				}
			}
		}
	}
	return results, nil
}

// table prints each case's runs by agent and arm: how many passed, and
// pass^k, whether every one of its k runs did.
func table(out io.Writer, results []Result) {
	type key struct{ c, a, arm string }
	var order []key
	pass, all := map[key]int{}, map[key]int{}
	for _, r := range results {
		k := key{r.Case, r.Agent, r.Arm}
		if all[k] == 0 {
			order = append(order, k)
		}
		all[k]++
		if r.Pass {
			pass[k]++
		}
	}
	for _, k := range order {
		yes := "no"
		if pass[k] == all[k] {
			yes = "yes"
		}
		fmt.Fprintf(out, "%s  %s  %-7s  %d/%d  pass^%d %s\n", k.c, k.a, k.arm, pass[k], all[k], all[k], yes)
	}
	for _, r := range results {
		if !r.Pass {
			fmt.Fprintf(out, "  %s %s %s: %s\n", r.Case, r.Agent, r.Arm, r.Reason)
		}
	}
}

// report tells the board in dir of each run: a line in the eval log, and
// a point added to the charts. Never chart set: it would wipe the runs
// before.
func report(dir string, results []Result, x exe) error {
	for _, r := range results {
		line := fmt.Sprintf("case %s: PASS %s %s, %d cmds", r.Case, r.Agent, r.Arm, r.Cmds)
		point := "1"
		if !r.Pass {
			line = fmt.Sprintf("case %s: FAIL %s %s, %d cmds: %s", r.Case, r.Agent, r.Arm, r.Cmds, r.Reason)
			point = "0"
		}
		arm := r.Agent + "-" + r.Arm
		calls := [][]string{
			{"log", "add", "eval", "--time", line},
			{"chart", "add", "eval-pass-" + arm, r.Case, point},
		}
		if r.Use.Input > 0 {
			calls = append(calls,
				[]string{"chart", "add", "eval-tokens-" + arm, "input", fmt.Sprint(r.Use.Input)},
				[]string{"chart", "add", "eval-tokens-" + arm, "output", fmt.Sprint(r.Use.Output)})
		}
		for _, c := range calls {
			if err := x(dir, c...); err != nil {
				return err
			}
		}
	}
	return nil
}
