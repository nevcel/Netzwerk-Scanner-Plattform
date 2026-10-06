package service

import "testing"

func TestValidateTarget(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{
			name:    "Loopback-Adresse",
			target:  "127.0.0.1",
			wantErr: false,
		},
		{
			name:    "Private IPv4-Adresse",
			target:  "192.168.1.10",
			wantErr: false,
		},
		{
			name:    "Privates Netzwerk /24",
			target:  "192.168.1.0/24",
			wantErr: false,
		},
		{
			name:    "Leeres Ziel",
			target:  "",
			wantErr: true,
		},
		{
			name:    "Ungültige Eingabe",
			target:  "abc",
			wantErr: true,
		},
		{
			name:    "Öffentliche IPv4-Adresse",
			target:  "8.8.8.8",
			wantErr: true,
		},
		{
			name:    "Netzwerk grösser als /24",
			target:  "192.168.0.0/23",
			wantErr: true,
		},
		{
			name:    "IPv6-Adresse",
			target:  "::1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTarget(tt.target)

			if tt.wantErr && err == nil {
				t.Errorf(
					"ValidateTarget(%q) sollte einen Fehler liefern",
					tt.target,
				)
			}

			if !tt.wantErr && err != nil {
				t.Errorf(
					"ValidateTarget(%q) lieferte unerwarteten Fehler: %v",
					tt.target,
					err,
				)
			}
		})
	}
}
