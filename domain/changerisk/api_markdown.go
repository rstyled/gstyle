package changerisk

import "fmt"

func apiVerdict(api APIReport) string {
	return map[APIStatus]string{
		APIUnchanged: "unchanged ✅", APICompatible: "compatible changes ✅",
		APIBreaking: "breaking changes ⚠️", APIUnavailable: "unavailable ⚠️",
		APIPartial: "partial 👀",
	}[api.Status]
}

func apiSummary(api APIReport) string {
	if api.Status == APIUnavailable {
		return text(api.Error) + "."
	}
	if api.Status == APIPartial {
		return text(api.Limits[0]) + "."
	}
	if len(api.Changes) == 0 {
		return fmt.Sprintf("No exported API changes in %d root-module packages (current build).", api.PackagesAfter)
	}
	change := firstAPIChange(api.Changes)
	return fmt.Sprintf("Root module: %s: %s (%d API changes).", text(change.Package), text(change.Message), len(api.Changes))
}

func firstAPIChange(changes []APIChange) APIChange {
	for _, change := range changes {
		if !change.Compatible {
			return change
		}
	}
	return changes[0]
}
