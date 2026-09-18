package config

import (
	"testing"
)

type validatable struct {
	String string   `validate:"required"                yaml:"string"`
	Enum   string   `validate:"required,oneof=on off"   yaml:"enum"`
	Int    int      `validate:"omitempty,min=1"         yaml:"int"`
	List   []string `validate:"omitempty,dive,required" yaml:"list"`
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    validatable
		wantErr bool
	}{
		{name: "complete", give: validatable{String: "value", Enum: "on"}},
		{name: "missing required", give: validatable{String: "value"}, wantErr: true},
		{name: "value outside oneof", give: validatable{String: "value", Enum: "unknown"}, wantErr: true},
		{name: "optional omitted", give: validatable{String: "value", Enum: "off"}},
		{name: "optional below min", give: validatable{String: "value", Enum: "on", Int: -1}, wantErr: true},
		{name: "empty item in list", give: validatable{String: "value", Enum: "on", List: []string{""}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				if err := validate(tt.give); (err != nil) != tt.wantErr {
					t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			},
		)
	}
}
