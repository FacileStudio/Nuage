package appregistry

import "testing"

func TestParse(t *testing.T) {
	apps, err := Parse(" plume:plume-client:Plume ; ardoise:Ab12.cd_ef ;", "nuage-client")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []App{{Key: "plume", Name: "Plume", ClientID: "plume-client"}, {Key: "ardoise", Name: "ardoise", ClientID: "Ab12.cd_ef"}}
	if len(apps) != len(want) || apps[0] != want[0] || apps[1] != want[1] {
		t.Fatalf("Parse = %+v, want %+v", apps, want)
	}
	for _, raw := range []string{
		"plume",
		"Plume:client",
		"plume:bad client",
		"plume:a;plume:b",
		"plume:a;ardoise:a",
		"plume:nuage-client",
		"plume:a:" + string(rune(1)),
	} {
		if _, err := Parse(raw, "nuage-client"); err == nil {
			t.Errorf("Parse(%q) accepted a malformed value", raw)
		}
	}
}
