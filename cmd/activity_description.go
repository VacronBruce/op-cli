package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/viper"
)

type diffSource interface {
	DescriptionDiff(journalID int) ([]api.DiffSegment, error)
}

// newDiffSource and keychainPassword are seams so tests never hit a real
// server or the Keychain.
var newDiffSource = func() (diffSource, error) {
	user, password, err := webCredentials()
	if err != nil {
		return nil, err
	}
	return api.LoginWeb(viper.GetString("url"), user, password)
}

var keychainPassword = func(user string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", errors.New("keychain lookup is only available on macOS")
	}
	out, err := exec.Command("security", "find-generic-password", "-s", "op-cli-web", "-a", user, "-w").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func webCredentials() (user, password string, err error) {
	user = viper.GetString("web_user")
	if user == "" {
		return "", "", errors.New("web login needs a user: set web_user in ~/.oprc (or OP_WEB_USER)")
	}
	if password = viper.GetString("web_password"); password != "" {
		return user, password, nil
	}
	if password, err = keychainPassword(user); err != nil || password == "" {
		return "", "", fmt.Errorf("web login needs a password for %s: run `security add-generic-password -U -s op-cli-web -a %s -w` or set OP_WEB_PASSWORD", user, user)
	}
	return user, password, nil
}

type descriptionEdit struct {
	JournalID int    `json:"journal_id"`
	CreatedAt string `json:"created_at"`
	Author    string `json:"author"`
	Before    string `json:"before"`
	After     string `json:"after"`

	when     string
	wordDiff []string
}

func printDescriptionHistory(ac *api.ActivityCollection, asJSON bool) error {
	journals := api.DescriptionJournals(ac)
	if len(journals) == 0 {
		if asJSON {
			fmt.Println("[]")
		} else {
			fmt.Println("No description changes.")
		}
		return nil
	}
	src, err := newDiffSource()
	if err != nil {
		return err
	}
	edits := make([]descriptionEdit, 0, len(journals))
	for _, j := range journals {
		segs, err := src.DescriptionDiff(j.JournalID)
		if err != nil {
			return err
		}
		when := j.Activity.CreatedAt
		if t, err := time.Parse(time.RFC3339, when); err == nil {
			when = t.UTC().Format("2006-01-02 15:04 UTC")
		}
		edits = append(edits, descriptionEdit{
			JournalID: j.JournalID,
			CreatedAt: j.Activity.CreatedAt,
			Author:    j.Activity.Links.User.Title,
			Before:    api.Before(segs),
			After:     api.After(segs),
			when:      when,
			wordDiff:  api.WordDiffLines(segs),
		})
	}
	if asJSON {
		b, err := json.MarshalIndent(edits, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	for i, e := range edits {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("[%s] %s (journal #%d)\n", e.when, e.Author, e.JournalID)
		for _, l := range e.wordDiff {
			fmt.Printf("  %s\n", l)
		}
	}
	return nil
}
