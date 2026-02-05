package dictionary

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

const baseURL = "https://api.dictionaryapi.dev/api/v2/entries/en"

var client = &http.Client{Timeout: 10 * time.Second}

// Definition is LLM-friendly definition output
type Definition struct {
	Word          string    `json:"word"`
	Phonetic      string    `json:"phonetic,omitempty"`
	Origin        string    `json:"origin,omitempty"`
	Meanings      []Meaning `json:"meanings"`
	AudioURL      string    `json:"audio,omitempty"`
}

// Meaning groups definitions by part of speech
type Meaning struct {
	PartOfSpeech string   `json:"pos"`
	Definitions  []Def    `json:"definitions"`
	Synonyms     []string `json:"synonyms,omitempty"`
	Antonyms     []string `json:"antonyms,omitempty"`
}

// Def is a single definition
type Def struct {
	Definition string `json:"def"`
	Example    string `json:"example,omitempty"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dictionary",
		Aliases: []string{"dict", "define"},
		Short:   "Dictionary commands",
	}

	cmd.AddCommand(newDefineCmd())
	cmd.AddCommand(newSynonymsCmd())
	cmd.AddCommand(newAntonymsCmd())

	return cmd
}

func newDefineCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "define [word]",
		Short: "Get word definition",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			word := strings.ToLower(args[0])

			entries, err := fetchWord(word)
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				return output.PrintError("not_found", "Word not found: "+word, nil)
			}

			entry := entries[0]
			def := Definition{
				Word:     entry.Word,
				Phonetic: entry.Phonetic,
				Origin:   entry.Origin,
			}

			// Get audio URL
			for _, p := range entry.Phonetics {
				if p.Audio != "" {
					def.AudioURL = p.Audio
					break
				}
			}

			// Process meanings
			for _, m := range entry.Meanings {
				meaning := Meaning{
					PartOfSpeech: m.PartOfSpeech,
				}

				// Collect synonyms/antonyms from meaning level
				if len(m.Synonyms) > 0 {
					meaning.Synonyms = limitSlice(m.Synonyms, 5)
				}
				if len(m.Antonyms) > 0 {
					meaning.Antonyms = limitSlice(m.Antonyms, 5)
				}

				// Process definitions
				defCount := limit
				if defCount > len(m.Definitions) {
					defCount = len(m.Definitions)
				}

				for i := 0; i < defCount; i++ {
					d := m.Definitions[i]
					meaning.Definitions = append(meaning.Definitions, Def{
						Definition: d.Definition,
						Example:    d.Example,
					})

					// Also collect synonyms/antonyms from definition level
					if len(d.Synonyms) > 0 && len(meaning.Synonyms) < 5 {
						for _, s := range d.Synonyms {
							if len(meaning.Synonyms) >= 5 {
								break
							}
							if !contains(meaning.Synonyms, s) {
								meaning.Synonyms = append(meaning.Synonyms, s)
							}
						}
					}
				}

				def.Meanings = append(def.Meanings, meaning)
			}

			return output.Print(def)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 3, "Max definitions per part of speech")

	return cmd
}

func newSynonymsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "synonyms [word]",
		Aliases: []string{"syn"},
		Short:   "Get synonyms for a word",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			word := strings.ToLower(args[0])

			entries, err := fetchWord(word)
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				return output.PrintError("not_found", "Word not found: "+word, nil)
			}

			synonyms := make(map[string][]string)

			for _, entry := range entries {
				for _, m := range entry.Meanings {
					pos := m.PartOfSpeech

					// Collect from meaning level
					for _, s := range m.Synonyms {
						if !contains(synonyms[pos], s) {
							synonyms[pos] = append(synonyms[pos], s)
						}
					}

					// Collect from definition level
					for _, d := range m.Definitions {
						for _, s := range d.Synonyms {
							if !contains(synonyms[pos], s) {
								synonyms[pos] = append(synonyms[pos], s)
							}
						}
					}
				}
			}

			if len(synonyms) == 0 {
				return output.PrintError("not_found", "No synonyms found for: "+word, nil)
			}

			// Flatten to simple output
			type Result struct {
				Word     string              `json:"word"`
				Synonyms map[string][]string `json:"synonyms"`
			}

			return output.Print(Result{
				Word:     word,
				Synonyms: synonyms,
			})
		},
	}

	return cmd
}

func newAntonymsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "antonyms [word]",
		Aliases: []string{"ant"},
		Short:   "Get antonyms for a word",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			word := strings.ToLower(args[0])

			entries, err := fetchWord(word)
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				return output.PrintError("not_found", "Word not found: "+word, nil)
			}

			antonyms := make(map[string][]string)

			for _, entry := range entries {
				for _, m := range entry.Meanings {
					pos := m.PartOfSpeech

					// Collect from meaning level
					for _, a := range m.Antonyms {
						if !contains(antonyms[pos], a) {
							antonyms[pos] = append(antonyms[pos], a)
						}
					}

					// Collect from definition level
					for _, d := range m.Definitions {
						for _, a := range d.Antonyms {
							if !contains(antonyms[pos], a) {
								antonyms[pos] = append(antonyms[pos], a)
							}
						}
					}
				}
			}

			if len(antonyms) == 0 {
				return output.PrintError("not_found", "No antonyms found for: "+word, nil)
			}

			type Result struct {
				Word     string              `json:"word"`
				Antonyms map[string][]string `json:"antonyms"`
			}

			return output.Print(Result{
				Word:     word,
				Antonyms: antonyms,
			})
		},
	}

	return cmd
}

type apiEntry struct {
	Word      string `json:"word"`
	Phonetic  string `json:"phonetic"`
	Origin    string `json:"origin"`
	Phonetics []struct {
		Text  string `json:"text"`
		Audio string `json:"audio"`
	} `json:"phonetics"`
	Meanings []struct {
		PartOfSpeech string   `json:"partOfSpeech"`
		Synonyms     []string `json:"synonyms"`
		Antonyms     []string `json:"antonyms"`
		Definitions  []struct {
			Definition string   `json:"definition"`
			Example    string   `json:"example"`
			Synonyms   []string `json:"synonyms"`
			Antonyms   []string `json:"antonyms"`
		} `json:"definitions"`
	} `json:"meanings"`
}

func fetchWord(word string) ([]apiEntry, error) {
	reqURL := fmt.Sprintf("%s/%s", baseURL, url.PathEscape(word))

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, output.PrintError("fetch_failed", err.Error(), nil)
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, output.PrintError("fetch_failed", err.Error(), nil)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, output.PrintError("not_found", "Word not found: "+word, nil)
	}

	if resp.StatusCode >= 400 {
		return nil, output.PrintError("fetch_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	var entries []apiEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, output.PrintError("parse_failed", err.Error(), nil)
	}

	return entries, nil
}

func limitSlice(s []string, max int) []string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
