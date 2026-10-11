package cobra

import (
	"context"
	"fmt"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/spf13/cobra"
	"os"
	"sort"
	"strconv"
	"time"
)

type CobraRuntime struct {
	root *cobra.Command
}

type CobraRuntimeConfig struct {
	Name    string
	Short   string
	Long    string
	Version string
}

// NewCobraRuntime construit le runtime Cobra (root command)
func NewCobraRuntime(cfg CobraRuntimeConfig) *CobraRuntime {
	root := &cobra.Command{
		Use:     cfg.Name,
		Short:   cfg.Short,
		Long:    cfg.Long,
		Version: cfg.Version,
	}

	root.CompletionOptions.DisableDefaultCmd = true

	return &CobraRuntime{root: root}
}

func (r *CobraRuntime) AddCommand(cmd ru.Command) {

	c := &cobra.Command{
		Use:   cmd.Command,
		Short: cmd.Short,
		Long:  cmd.Long,
	}

	declareFlags(c, cmd.Args)

	c.RunE = func(cobraCmd *cobra.Command, _ []string) error {
		parsed, err := parseFlagsToParsedArgs(cobraCmd, cmd.Args)
		if err != nil {
			return err
		}
		ctx := cobraCmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		return cmd.Exec(ctx, parsed)
	}

	r.root.AddCommand(c)
}

func (r *CobraRuntime) Process(ctx context.Context) error {
	r.root.SetContext(ctx)
	return r.root.Execute()
}

/* ----------------------- Flags: declaration ----------------------- */

// declareFlags registers every flag of a stage on its cobra command.
//
// It panics on a spec the runtime cannot honour — an unknown flag type, or a
// default value that cannot be read as the flag's type. Both are programming
// errors in the stage's Configure, and a stage whose flags are wrong cannot run
// anyway: better at startup than as a wrong value silently passed to the stage.
func declareFlags(c *cobra.Command, specs []ru.Flag) {
	for _, s := range specs {

		value := flagDefault{flag: s}

		switch s.Type {
		case ru.StringFlag:
			c.Flags().String(s.Name, value.asString(), s.Description)
		case ru.BoolFlag:
			c.Flags().Bool(s.Name, value.asBool(), s.Description)
		case ru.IntFlag:
			c.Flags().Int(s.Name, value.asInt(), s.Description)
		case ru.DurationFlag:
			c.Flags().Duration(s.Name, value.asDuration(), s.Description)
		default:
			panic(fmt.Sprintf("flag %q: unknown flag type %q", s.Name, s.Type))
		}

		if s.Required {
			_ = c.MarkFlagRequired(s.Name)
		}
	}
}

// flagDefault reads a flag's declared default into the type the flag needs.
// It accepts both a typed value and its string form, so that AddFlag("jobs",
// ru.IntFlag, ..., 4, ...) and AddFlag("jobs", ru.IntFlag, ..., "4", ...) both
// yield 4 — and a value that cannot be read at all is reported, not dropped.
type flagDefault struct {
	flag ru.Flag
}

func (d flagDefault) asString() string {
	if d.flag.Default == nil {
		return ""
	}
	if s, ok := d.flag.Default.(string); ok {
		return s
	}
	return fmt.Sprint(d.flag.Default)
}

func (d flagDefault) asBool() bool {
	switch v := d.flag.Default.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		if v == "" {
			return false
		}
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			panic(d.unreadable("bool", err))
		}
		return parsed
	default:
		panic(d.unreadable("bool", nil))
	}
}

func (d flagDefault) asInt() int {
	switch v := d.flag.Default.(type) {
	case nil:
		return 0
	case int:
		return v
	case string:
		if v == "" {
			return 0
		}
		parsed, err := strconv.Atoi(v)
		if err != nil {
			panic(d.unreadable("int", err))
		}
		return parsed
	default:
		panic(d.unreadable("int", nil))
	}
}

func (d flagDefault) asDuration() time.Duration {
	switch v := d.flag.Default.(type) {
	case nil:
		return 0
	case time.Duration:
		return v
	case string:
		if v == "" {
			return 0
		}
		parsed, err := time.ParseDuration(v)
		if err != nil {
			panic(d.unreadable("duration", err))
		}
		return parsed
	default:
		panic(d.unreadable("duration", nil))
	}
}

func (d flagDefault) unreadable(want string, cause error) string {
	if cause != nil {
		return fmt.Sprintf("flag %q: default %#v is not a valid %s: %v", d.flag.Name, d.flag.Default, want, cause)
	}
	return fmt.Sprintf("flag %q: default %#v cannot be read as %s", d.flag.Name, d.flag.Default, want)
}

/* ----------------------- Flags: parsing -> ParsedArgs ----------------------- */

func parseFlagsToParsedArgs(c *cobra.Command, specs []ru.Flag) (ru.ParsedArgs, error) {

	args := ru.ParsedArgs{}

	for _, s := range specs {
		switch s.Type {
		case ru.StringFlag:
			v, err := c.Flags().GetString(s.Name)
			if err != nil {
				return ru.ParsedArgs{}, err
			}
			args.SetString(s.Name, v)

		case ru.BoolFlag:
			v, err := c.Flags().GetBool(s.Name)
			if err != nil {
				return ru.ParsedArgs{}, err
			}
			args.SetBool(s.Name, v)

		case ru.IntFlag:
			v, err := c.Flags().GetInt(s.Name)
			if err != nil {
				return ru.ParsedArgs{}, err
			}
			args.SetInt(s.Name, v)

		case ru.DurationFlag:
			v, err := c.Flags().GetDuration(s.Name)
			if err != nil {
				return ru.ParsedArgs{}, err
			}
			args.SetDuration(s.Name, v)

		default:
			return ru.ParsedArgs{}, fmt.Errorf("unknown flag type for %s: %s", s.Name, s.Type)
		}
	}

	return args, nil
}

func (r *CobraRuntime) AddListCommand() {
	c := &cobra.Command{
		Use:   "list",
		Short: "List available commands",
		Run: func(cmd *cobra.Command, _ []string) {
			var names []string
			for _, sc := range r.root.Commands() {
				if sc.Name() == "help" || sc.Name() == "list" {
					continue
				}
				names = append(names, sc.Name())
			}
			sort.Strings(names)
			for _, n := range names {
				fmt.Fprintln(os.Stdout, n)
			}
		},
	}
	r.root.AddCommand(c)
}
