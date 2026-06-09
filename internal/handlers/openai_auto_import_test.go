package handlers

import "testing"

func TestDetectAutoImportFormat(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		raw      string
		want     string
	}{
		{
			name: "easyllm export",
			raw:  `{"oauth_accounts":[{"email":"a@b.com","access_token":"at"}],"api_accounts":[]}`,
			want: scanFormatEasyLLMExport,
		},
		{
			name:     "cpa by filename",
			filename: "user@example.com-cpa.json",
			raw:      `{"type":"codex","email":"u@example.com","access_token":"at","refresh_token":"rt"}`,
			want:     scanFormatCPA,
		},
		{
			name: "token flat",
			raw:  `{"email":"a@b.com","access_token":"at","refresh_token":"rt","id_token":"id"}`,
			want: scanFormatToken,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := detectAutoImportFormat([]byte(tc.raw), tc.filename)
			if got != tc.want {
				t.Fatalf("detectAutoImportFormat() = %q, want %q", got, tc.want)
			}
		})
	}
}
