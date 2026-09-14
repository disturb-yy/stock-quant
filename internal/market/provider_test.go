package market

import "testing"

func TestSelectProvider(t *testing.T) {
	tests := []struct {
		name         string
		requested    string
		fixtureReady bool
		wantMode     ProviderMode
		wantProvider string
	}{
		{name: "ready demo fixture", requested: "demo", fixtureReady: true, wantMode: ModeDemo, wantProvider: DemoProviderName},
		{name: "unseeded local fallback", requested: "demo", fixtureReady: false, wantMode: ModeFallback, wantProvider: FallbackProviderName},
		{name: "unavailable real provider falls back", requested: "real", fixtureReady: true, wantMode: ModeFallback, wantProvider: FallbackProviderName},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := SelectProvider(test.requested, test.fixtureReady)
			if got.Mode != test.wantMode {
				t.Fatalf("mode = %q, want %q", got.Mode, test.wantMode)
			}
			if got.Provider != test.wantProvider {
				t.Fatalf("provider = %q, want %q", got.Provider, test.wantProvider)
			}
		})
	}
}
