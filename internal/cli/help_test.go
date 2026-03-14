package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootAndKeyCommandsExposeWorkflowExamples(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		args []string
		want []string
	}{
		{
			args: []string{"--help"},
			want: []string{
				"sandnote init",
				"sandnote artifact import ./spec.md --id art_spec --mode reference",
				"sandnote resume",
				"sandnote repl",
			},
		},
		{
			args: []string{"artifact", "import", "--help"},
			want: []string{
				"sandnote artifact import ./diagd-spec.md --id art_diagd --mode reference",
				"--entry en_auth",
			},
		},
		{
			args: []string{"thread", "checkpoint", "--help"},
			want: []string{
				"sandnote thread checkpoint th_auth",
				"--reentry-anchor en_auth",
			},
		},
		{
			args: []string{"topic", "promote", "--help"},
			want: []string{
				"sandnote topic promote tp_auth --thread th_auth",
				"--include-supporting",
			},
		},
	} {
		cmd := NewRootCommand()
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		cmd.SetErr(output)
		cmd.SetArgs(tc.args)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute(%v) error = %v\noutput=%s", tc.args, err, output.String())
		}
		text := output.String()
		for _, want := range tc.want {
			if !strings.Contains(text, want) {
				t.Fatalf("help output for %v missing %q:\n%s", tc.args, want, text)
			}
		}
	}
}
