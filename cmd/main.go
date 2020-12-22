package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
	"go.uber.org/multierr"

	"github.com/mlowery/kworx"
)

func main() {
	app := &cli.App{
		Name:  "🏋️kworx",
		Usage: "Multi-threaded kubectl",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Run templated command in parallel against single cluster.",
				Action: func(c *cli.Context) error {
					var cmd string
					var args []string
					for _, arg := range c.Args().Slice() {
						if cmd == "" {
							cmd = arg
							continue
						}
						if cmd != "" {
							args = append(args, arg)
						}
					}
					if cmd == "" {
						log.Fatalf("command is required")
					}
					b, err := ioutil.ReadFile(c.String("values-file"))
					if err != nil {
						log.Fatalf("failed to read values-file %q: %v", c.String("values-file"), err)
					}
					values := strings.Split(string(b), "\n")
					runner := kworx.NewRunner(c.Int("workers"), values, kworx.NewCommandFunc(c.String("output"), cmd, args...))
					sigCh := make(chan os.Signal)
					stopCh := make(chan struct{})
					signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
					go func() {
						<-sigCh
						log.Println("shutting down")
						stopCh <- struct{}{}
					}()
					err = runner.Run(stopCh)
					if err != nil {
						errs := multierr.Errors(err)
						var buffer bytes.Buffer
						for i, err := range errs {
							buffer.WriteString(fmt.Sprintf("[error %3d]: %v\n", i+1, err))
						}
						log.Fatalf("failed to run (%d errors):\n%s", len(errs), buffer.String())
					}
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "values-file", Usage: "path to newline-separated values to pass to command", Required: true},
					&cli.IntFlag{Name: "workers", Aliases: []string{"w"}, Value: 10, Usage: "level of parallelism"},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: kworx.CommandFuncOutputColor, Usage: fmt.Sprintf("output type (one of %s)", strings.Join(kworx.CommandFuncOutputOptions, ","))},
				},
			},
		},
		CommandNotFound: func(c *cli.Context, command string) {
			fmt.Fprintf(c.App.Writer, "unknown command: %q\n", command)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
