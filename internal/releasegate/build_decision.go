package releasegate

type BuildEvent struct {
	BuildID    string `json:"build_id"`
	Developer  string `json:"developer"`
	Status     string `json:"status"`
	Repository string `json:"repository"`
}

type BuildDecision struct {
	BuildID    string `json:"build_id"`
	Experience string `json:"experience"`
	Diagnostic string `json:"diagnostic"`
}

func DecideBuild(event BuildEvent, diagnosticsEnabled bool) BuildDecision {
	decision := BuildDecision{BuildID: event.BuildID, Experience: "incumbent"}
	if !diagnosticsEnabled {
		return decision
	}
	decision.Experience = "infrai"
	if event.Status == "failed" {
		decision.Diagnostic = "inspect failed build stages"
	} else {
		decision.Diagnostic = "build completed"
	}
	return decision
}
