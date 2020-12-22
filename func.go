package kworx

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

var (
	// pointer since Mutex cannot be copied
	logMutex = &sync.Mutex{}

	colorWheel = []*color.Color{
		color.New(color.FgRed),
		color.New(color.FgGreen),
		color.New(color.FgYellow),
		color.New(color.FgBlue),
		color.New(color.FgMagenta),
		color.New(color.FgCyan),
	}

	colorIndex = 0

	CommandFuncOutputOptions = []string{
		CommandFuncOutputPlain,
		CommandFuncOutputColor,
		CommandFuncOutputPrefix,
		CommandFuncOutputNone,
	}

	newLineRegexp = regexp.MustCompile(`\n`)
)

const (
	CommandFuncOutputPlain  = "plain"
	CommandFuncOutputColor  = "color"
	CommandFuncOutputPrefix = "prefix"
	CommandFuncOutputNone   = "none"
)

func NewCommandFunc(output string, name string, arg ...string) DoWithValueFunc {
	return func(value string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, name, arg...)
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("KWORX_VALUE=%s", value),
		)
		stdoutStderr, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to run command %q: %s: %w", name, stdoutStderr, err)
		}
		if output == CommandFuncOutputNone {
			return nil
		}
		outString := string(stdoutStderr)
		// remove it now so it can be added later
		outString = strings.TrimSuffix(outString, "\n")
		if len(outString) == 0 {
			return nil
		}
		// lock ensures no interleaving of output and protects colorIndex
		logMutex.Lock()
		defer logMutex.Unlock()
		printf := func(format string, a ...interface{}) {
			fmt.Printf(format, a...)
		}
		switch output {
		case CommandFuncOutputColor:
			if colorIndex == len(colorWheel)-1 {
				colorIndex = 0
			}
			printf = colorWheel[colorIndex].PrintfFunc()
			colorIndex += 1
			// color mode also gets the prefix so fall through
			fallthrough
		case CommandFuncOutputPrefix:
			linePrefix := fmt.Sprintf("%50s|", value)
			if len(outString) > 0 {
				outString = linePrefix + newLineRegexp.ReplaceAllString(outString, "\n"+linePrefix)
			}
		}
		if len(outString) > 0 {
			printf("%s\n", outString)
		}
		return nil
	}
}
