package main

import (
	"flag"
	"testing"
)

// TestValidateArgumentsAcceptsNDLSpecificSearchOnly は、NDL固有検索条件だけの検索を受け付けることを確認する
func TestValidateArgumentsAcceptsNDLSpecificSearchOnly(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
	}{
		{name: "from", args: []string{"2024", "", "", ""}},
		{name: "to", args: []string{"", "2024", "", ""}},
		{name: "subject", args: []string{"", "", "動物", ""}},
		{name: "description", args: []string{"", "", "", "ハムテル"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateArguments(nil, "", "", "", "", test.args[0], test.args[1], test.args[2], test.args[3], false); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestValidateArgumentsRejectsExplicitSearchFlagsWithISBN は、ISBN参照と検索用フラグの併用を拒否することを確認する
func TestValidateArgumentsRejectsExplicitSearchFlagsWithISBN(t *testing.T) {
	if err := validateArguments([]string{"9784098627417"}, "", "", "", "", "", "", "", "", true); err == nil {
		t.Fatal("explicit search flags with ISBN were accepted")
	}
}

// TestSearchOptionWasSetRecognizesNDLFlags は、NDL固有条件と漫画分類フラグを検索用フラグとして扱うことを確認する
func TestSearchOptionWasSetRecognizesNDLFlags(t *testing.T) {
	originalCommandLine := flag.CommandLine
	t.Cleanup(func() { flag.CommandLine = originalCommandLine })
	for _, name := range []string{"from", "to", "subject", "description", "manga-ndc", "manga-ndlc"} {
		t.Run(name, func(t *testing.T) {
			flags := flag.NewFlagSet("test", flag.ContinueOnError)
			flags.String(name, "", "")
			if err := flags.Set(name, "value"); err != nil {
				t.Fatal(err)
			}
			flag.CommandLine = flags
			if !searchOptionWasSet() {
				t.Fatalf("%s was not recognized as a search option", name)
			}
		})
	}
}
