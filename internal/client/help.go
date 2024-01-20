package client

/*
 * help.go
 * Help command handler
 * By J. Stuart McMurray
 * Created 20231218
 * Last Modified 20240120
 */

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
	"text/tabwriter"

	"golang.org/x/exp/maps"
	"golang.org/x/tools/txtar"
)

// topicTexts contains the help topics as a txtar archive.
//
//go:embed help.txtar
var topicTxtar []byte

// topicsTopic is the special topic which lists the help topics.
const topicsTopic = "topics"

// helpTopic contains the help text for a given topic.
type helpTopic struct {
	Description string
	Text        string
}

// helpTopics contains the help topics by name.  This is set by init() and
// should be treated as read-only.
var helpTopics map[string]helpTopic

// init un txtar's topicTxtar into helpTopics.
func init() {
	t := make(map[string]helpTopic)
	if err := unTxtarHelp(t, topicTxtar); nil != err {
		panic(fmt.Sprintf("parsing help topics txtar: %s", err))
	}
	helpTopics = t
}

// unTxtarHelp unpacks the txtar archive in tb into topics.
func unTxtarHelp(topics map[string]helpTopic, tb []byte) error {
	/* Unpack the archive. */
	for _, f := range txtar.Parse(tb).Files {
		/* Get the description and text. */
		d, t, ok := strings.Cut(string(f.Data), "\n")
		if !ok {
			return fmt.Errorf(
				"topic %s lacks separate description and text",
				f.Name,
			)
		}
		/* Tidy up and save it. */
		topics[f.Name] = helpTopic{
			Description: strings.TrimSpace(d),
			Text:        strings.TrimSpace(t),
		}
	}

	return nil
}

// helpHandler lists the available commands.
func helpHandler(s shell, name, args []string) error {
	/* If we've got a request for a topic, print it out. */
	if 0 != len(args) {
		printTopic(s, args[0])
		return nil
	}

	/* Print a bit of generic help. */
	s.Printf("Available Commands:\n\n%s\n", s.Cdr().Table())
	s.Printf("%s", "\n"+strings.Trim(`
To get started:

1. ,list to see available implants.
2. ,seti <ID> to choose an implant.
3.  Anything not one of the above commands will be queued as implant tasking.
4. ,logs to return to watching Plonk's logs.
`, "\n")+"\n")
	return nil
}

// printTopic prints information about a help topic to the shell.
func printTopic(s shell, topic string) {
	/* If we're getting help for help, print the list. */
	if topic == topicsTopic {
		s.Printf("Here's what we know:\n\n")
		/* Make a nice sorted table of help topic descriptions. */
		ts := maps.Keys(helpTopics)
		slices.SortFunc(ts, func(a, b string) int {
			return strings.Compare(
				strings.ToLower(a),
				strings.ToLower(b),
			)
		})
		tw := tabwriter.NewWriter(s, 0, 8, 1, ' ', 0)
		defer tw.Flush()
		for _, t := range ts {
			fmt.Fprintf(
				tw,
				"%s\t- %s\n",
				t,
				helpTopics[t].Description,
			)
		}
		return
	}

	/* Try to get the help requested. */
	h, ok := helpTopics[topic]
	if !ok {
		s.ErrorLogf("Sorry, haven't heard of %q before :(", topic)
		return
	}
	s.Printf("%s\n", strings.Trim(h.Text, "\n"))
}
