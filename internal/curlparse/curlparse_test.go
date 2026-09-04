package curlparse

import "testing"

func TestParse(t *testing.T) {
	text := `curl 'https://s01.company.talknote.com/NetworkCode/ajax/feed/list' -H 'cookie: foo=1; TALKNOTE_SID2=secret%2Bvalue; talknote_user=x'`
	got, err := Parse(text)
	if err != nil {
		t.Fatal(err)
	}
	if got.Origin != "https://s01.company.talknote.com" || got.SID != "secret%2Bvalue" {
		t.Fatalf("got=%+v", got)
	}
}

func TestParseRejectsMissingValues(t *testing.T) {
	for _, text := range []string{
		`curl 'https://example.com/x' -b 'TALKNOTE_SID2=x'`,
		`curl 'https://s01.company.talknote.com/x' -b 'other=x'`,
	} {
		if _, err := Parse(text); err == nil {
			t.Fatalf("Parse(%q) succeeded", text)
		}
	}
}
