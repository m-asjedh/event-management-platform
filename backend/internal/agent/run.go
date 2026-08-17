package agent

import (
	"fmt"
	"io"
	"strings"
)

// Outcome is the transcript of one question. Tests assert Calls and Answer.
type Outcome struct {
	Calls  []Call
	Answer string
}

// Run maps a question to GET calls, prints each call and a short summary,
// then answers in a sentence. It never POSTs or PATCHes.
func Run(w io.Writer, client *Client, question string) (Outcome, error) {
	var hist []Observation
	var out Outcome

	for step := 0; step < maxSteps; step++ {
		act := next(question, hist)
		if act.finish {
			out.Answer = act.answer
			fmt.Fprintf(w, "\nAnswer: %s\n", out.Answer)
			return out, nil
		}
		obs, err := dispatch(client, act)
		if err != nil {
			return out, err
		}
		hist = append(hist, obs)
		out.Calls = append(out.Calls, obs.Call)
		fmt.Fprintf(w, "%s\n  %s\n", obs.Call, obs.Summary)
		if obs.Status >= 400 {
			body := strings.TrimSpace(obs.Body)
			if len(body) > 200 {
				body = body[:200] + "…"
			}
			if body != "" {
				fmt.Fprintf(w, "  %s\n", body)
			}
		}
	}

	out.Answer = "I stopped after 8 steps. I will not summarise work I did not finish."
	fmt.Fprintf(w, "\nAnswer: %s\n", out.Answer)
	return out, nil
}
