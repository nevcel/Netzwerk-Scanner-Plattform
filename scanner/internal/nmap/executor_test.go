package nmap

import "testing"

func TestValidateTarget(t *testing.T) {
	executor := NewExecutor()

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
			name:    "Privates 10er-Netz",
			target:  "10.0.0.10",
			wantErr: false,
		},
		{
			name:    "Privates Netzwerk /24",
			target:  "192.168.1.0/24",
			wantErr: false,
		},
		{
			name:    "Privates Netzwerk /25",
			target:  "192.168.1.0/25",
			wantErr: false,
		},
		{
			name:    "Leeres Ziel",
			target:  "",
			wantErr: true,
		},
		{
			name:    "Ungültiger Text",
			target:  "kein-netzwerk",
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
			err := executor.ValidateTarget(tt.target)

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
