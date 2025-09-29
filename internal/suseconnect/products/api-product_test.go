package products

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOperatorProduct_Methods(t *testing.T) {
	type fields struct {
		Identifier string
		Version    string
		Arch       string
	}
	tests := []struct {
		name       string
		fields     fields
		want       string
		wantValues []string
	}{
		{
			name: "basic test",
			fields: fields{
				Identifier: "Rancher Prime",
				Version:    "1.0.0",
				Arch:       "x86_64",
			},
			want: "Rancher Prime/1.0.0/x86_64",
			wantValues: []string{
				"Rancher Prime",
				"1.0.0",
				"x86_64",
			},
		},
		{
			name: "remove version prefix test",
			fields: fields{
				Identifier: "Rancher Prime",
				Version:    "v1.0.0",
				Arch:       "x86_64",
			},
			want: "Rancher Prime/1.0.0/x86_64",
			wantValues: []string{
				"Rancher Prime",
				"1.0.0",
				"x86_64",
			},
		},
	}

	for _, tt := range tests {
		t.Run("ToTriplet_"+tt.name, func(t *testing.T) {
			t.Parallel()
			op := OperatorProduct{
				Identifier: tt.fields.Identifier,
				Version:    tt.fields.Version,
				Arch:       tt.fields.Arch,
			}
			assert.Equalf(t, tt.want, op.ToTriplet(), "ToTriplet()")
		})

		t.Run("GetTripletValues_"+tt.name, func(t *testing.T) {
			t.Parallel()
			op := OperatorProduct{
				Identifier: tt.fields.Identifier,
				Version:    tt.fields.Version,
				Arch:       tt.fields.Arch,
			}
			assert.Len(t, tt.wantValues, 3, "Expected values for GetTripletValues must be 3 long")
			got, got1, got2 := op.GetTripletValues()
			assert.Equalf(t, tt.wantValues[0], got, "GetTripletValues()")
			assert.Equalf(t, tt.wantValues[1], got1, "GetTripletValues()")
			assert.Equalf(t, tt.wantValues[2], got2, "GetTripletValues()")
		})
	}
}

func TestOperatorProduct_sccSafeVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "1.0.0",
			want:  "1.0.0",
		},
		{
			input: "v1.0.0",
			want:  "1.0.0",
		},
		{
			input: "v2.12.1",
			want:  "2.12.1",
		},
		{
			input: "v2.12-dev-head-654324",
			want:  "2.12-dev-head-654324",
		},
		{
			input: "v2.12",
			want:  "2.12",
		},
		{
			input: "v2",
			want:  "2",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s => %s", tt.input, tt.want), func(t *testing.T) {
			op := OperatorProduct{
				Identifier: "Doesn't Matter",
				Version:    tt.input,
				Arch:       "Untested",
			}
			assert.Equalf(t, tt.want, op.sccSafeVersion(), "sccSafeVersion()")
		})
	}
}
