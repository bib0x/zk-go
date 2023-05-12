package cmd

import (
  "fmt"
  "io"
  "os"
  "github.com/mickael-menu/zk/internal/cli"
  "github.com/mickael-menu/zk/internal/util/errors"
  "github.com/mickael-menu/zk/internal/util/strings"
)

type Config struct { 
    List AliasList `cmd group:"cmd" default:"withargs" help:"List all current config aliases."`
}

type AliasList struct {
    Format     string   `group:format short:f placeholder:TEMPLATE   help:"Pretty print the list using a custom template or one of the predefined formats: name, full, json, jsonl."`
    Header     string   `group:format                                help:"Arbitrary text printed at the start of the list."`
    Footer     string   `group:format default:\n                     help:"Arbitrary text printed at the end of the list."`
    Delimiter  string   "group:format short:d default:\n             help:\"Print tags delimited by the given separator.\""  
    NoPager    bool     `group:format short:P help:"Do not pipe output into a pager."`
    Quiet      bool     `group:format short:q help:"Do not print the total number of tags found."`
}

func (cmd *AliasList) Run(container *cli.Container) error {
  cmd.Header = strings.ExpandWhitespaceLiterals(cmd.Header)
  cmd.Footer = strings.ExpandWhitespaceLiterals(cmd.Footer)
  cmd.Delimiter = strings.ExpandWhitespaceLiterals(cmd.Delimiter)
  
  if cmd.Format == "json" || cmd.Format == "jsonl" {
      if cmd.Header != "" {
          return errors.New("--header can't be used with JSON format")
      }
      if cmd.Footer != "\n" {
          return errors.New("--footer can't be used with JSON format")
      }
      if cmd.Delimiter != "\n" {
          return errors.New("--delimiter can't be used with JSON format")
      }

      switch cmd.Format {
      case "json":
          cmd.Delimiter = ","
          cmd.Header = "["
          cmd.Footer = "]\n"

      case "jsonl":
          // > The last character in the file may be a line separator, and it
          // > will be treated the same as if there was no line separator
          // > present.
          // > https://jsonlines.org/
          cmd.Footer = "\n"
      }
  }

  count := len(container.Config.Aliases)
  if count > 0 {
    err := container.Paginate(cmd.NoPager, func(out io.Writer) error {
          if cmd.Header != "" {
              fmt.Fprint(out, cmd.Header)
          }
          i := 0
          for key, value := range container.Config.Aliases {
              if (i > 0) {
                  fmt.Fprint(out, cmd.Delimiter)
              } 
              // Need custom formatter here
              alias := fmt.Sprintf("%s=%s", key, value)
              fmt.Fprint(out, alias)
              i++;
          }
          if cmd.Footer != "" {
              fmt.Fprint(out, cmd.Footer)
          }
          return nil;
      })
    return err;
  }

  if !cmd.Quiet {
    fmt.Fprintf(os.Stderr, "\nFound %d %s\n", count, strings.Pluralize("alias", count))
  }

  return nil; 
}

func (cmd *AliasList) aliasTemplate() string {
    format := cmd.Format
    if format == "" {
        format = "full"
    }

    templ, ok := map[string]string{
      "json":  `{{json .}}`,
      "jsonl": `{{json .}}`,
      "name":  `{{name}}`,
      "full":  `{{name}} ({{note-count}})`,
    }[format]
    if !ok {
        templ = strings.ExpandWhitespaceLiterals(format)
    }

    return templ
}
