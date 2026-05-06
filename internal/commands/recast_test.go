package commands

import (
	"reflect"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/foundry"
)

func TestMergeRecastOptions(t *testing.T) {
	cases := []struct {
		name     string
		recorded *foundry.CastOptionsRecord
		cli      recastCLIOptions
		want     foundry.CastOptionsRecord
	}{
		{
			name:     "empty recorded + empty CLI",
			recorded: nil,
			cli:      recastCLIOptions{},
			want:     foundry.CastOptionsRecord{},
		},
		{
			name:     "empty recorded + CLI flags",
			recorded: nil,
			cli: recastCLIOptions{
				WithWorkflows: true,
				ValueFiles:    []string{"./a.yaml"},
				SetOverrides:  []string{"k=v"},
			},
			want: foundry.CastOptionsRecord{
				WithWorkflows: true,
				ValueFiles:    []string{"./a.yaml"},
				SetOverrides:  []string{"k=v"},
			},
		},
		{
			name: "recorded + no CLI replays recorded",
			recorded: &foundry.CastOptionsRecord{
				WithWorkflows: true,
				ValueFiles:    []string{"./a.yaml"},
				SetOverrides:  []string{"k=v"},
			},
			cli: recastCLIOptions{},
			want: foundry.CastOptionsRecord{
				WithWorkflows: true,
				ValueFiles:    []string{"./a.yaml"},
				SetOverrides:  []string{"k=v"},
			},
		},
		{
			name:     "with-workflows is OR'd",
			recorded: &foundry.CastOptionsRecord{WithWorkflows: false},
			cli:      recastCLIOptions{WithWorkflows: true},
			want:     foundry.CastOptionsRecord{WithWorkflows: true},
		},
		{
			name:     "value files: recorded first, CLI appended",
			recorded: &foundry.CastOptionsRecord{ValueFiles: []string{"./a.yaml"}},
			cli:      recastCLIOptions{ValueFiles: []string{"./b.yaml"}},
			want:     foundry.CastOptionsRecord{ValueFiles: []string{"./a.yaml", "./b.yaml"}},
		},
		{
			name:     "value files: dedupe exact path",
			recorded: &foundry.CastOptionsRecord{ValueFiles: []string{"./a.yaml", "./b.yaml"}},
			cli:      recastCLIOptions{ValueFiles: []string{"./a.yaml", "./c.yaml"}},
			want:     foundry.CastOptionsRecord{ValueFiles: []string{"./a.yaml", "./b.yaml", "./c.yaml"}},
		},
		{
			name:     "set: distinct keys append",
			recorded: &foundry.CastOptionsRecord{SetOverrides: []string{"a=1"}},
			cli:      recastCLIOptions{SetOverrides: []string{"b=2"}},
			want:     foundry.CastOptionsRecord{SetOverrides: []string{"a=1", "b=2"}},
		},
		{
			name:     "set: same-key collision replaces recorded",
			recorded: &foundry.CastOptionsRecord{SetOverrides: []string{"a=1", "b=2"}},
			cli:      recastCLIOptions{SetOverrides: []string{"a=99"}},
			want:     foundry.CastOptionsRecord{SetOverrides: []string{"a=99", "b=2"}},
		},
		{
			name:     "set: nested-key collision uses dotted prefix",
			recorded: &foundry.CastOptionsRecord{SetOverrides: []string{"agent.target=foo"}},
			cli:      recastCLIOptions{SetOverrides: []string{"agent.target=bar"}},
			want:     foundry.CastOptionsRecord{SetOverrides: []string{"agent.target=bar"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeRecastOptions(tc.recorded, tc.cli)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %+v\nwant %+v", got, tc.want)
			}
		})
	}
}
