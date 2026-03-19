package cobra

import (
	"context"
	"fmt"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/spf13/cobra"
	"os"
	"sort"
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

	return &CobraRuntime{root: root}
}

func (r *CobraRuntime) AddCommand(cmd ru.Command) {
	c := &cobra.Command{
		Use:   cmd.Command,
		Short: cmd.Short,
		Long:  cmd.Long,
	}

	// 1) Déclare les flags à partir de Flag
	declareFlags(c, cmd.Args)

	// 2) RunE: parse flags -> ru.ParsedArgs -> call cmd.Exec
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

/* ----------------------- Flags: déclaration ----------------------- */

func declareFlags(c *cobra.Command, specs []ru.Flag) {
	for _, s := range specs {
		switch s.Type {
		case "string":
			def, _ := s.Default.(string)
			c.Flags().String(s.Name, def, s.Description)
		case "bool":
			def, _ := s.Default.(bool)
			c.Flags().Bool(s.Name, def, s.Description)
		case "int":
			def, _ := s.Default.(int)
			c.Flags().Int(s.Name, def, s.Description)
		default:
			// panic(fmt.Sprintf("unknown flag type: %s", s.Type))
			c.Flags().String(s.Name, fmt.Sprint(s.Default), s.Description)
		}

		if s.Required {
			_ = c.MarkFlagRequired(s.Name)
		}
	}
}

/* ----------------------- Flags: parsing -> ParsedArgs ----------------------- */

func parseFlagsToParsedArgs(c *cobra.Command, specs []ru.Flag) (ru.ParsedArgs, error) {
	// NOTE: il te faut un constructeur côté ru.
	// Exemple: ru.NewParsedArgs() qui initialise les maps.
	args := ru.ParsedArgs{}

	for _, s := range specs {
		switch s.Type {
		case "string":
			v, err := c.Flags().GetString(s.Name)
			if err != nil {
				return ru.ParsedArgs{}, err
			}
			// tu peux choisir de ne set que si Changed
			// if c.Flags().Changed(s.Name) { ... }
			args.SetString(s.Name, v)

		case "bool":
			v, err := c.Flags().GetBool(s.Name)
			if err != nil {
				return ru.ParsedArgs{}, err
			}
			args.SetBool(s.Name, v)

		case "int":
			v, err := c.Flags().GetInt(s.Name)
			if err != nil {
				return ru.ParsedArgs{}, err
			}
			args.SetInt(s.Name, v)

		default:
			return ru.ParsedArgs{}, fmt.Errorf("unknown flag type for %s: %s", s.Name, s.Type)
		}
	}

	// Optionnel: si tu veux rendre l’UX meilleure: afficher les flags connus si erreur
	// ou ordonner les args (help, etc.)
	_ = sort.Strings // just in case you add listing

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
