package resourceflow

// SummarizeFactories solves package-local first-result contracts. Invalid return
// dependencies are removed to a greatest fixed point. A separate least fixed
// point requires a path to an actual acquisition, so unseeded call cycles cannot
// invent a file. Failure-empty guarantees are solved independently.
func SummarizeFactories(queries map[string]*Query, helpers map[string]bool) map[string]FactorySummary {
	descriptors := map[string]factoryDescriptor{}
	for identity, query := range queries {
		descriptors[identity] = query.factory(helpers, nil)
	}
	possible := potentialFactories(descriptors)
	isolated := isolatedFactories(descriptors, possible)
	for identity, query := range queries {
		descriptors[identity] = query.factory(helpers, isolated)
	}
	valid := validFactories(descriptors, possible)
	failures := failureFactories(descriptors, valid, possible)
	summaries := make(map[string]FactorySummary, len(descriptors))
	for identity := range descriptors {
		summaries[identity] = FactorySummary{Fresh: valid[identity] && possible[identity], FailureEmpty: failures[identity]}
	}
	return summaries
}
func potentialFactories(descriptors map[string]factoryDescriptor) map[string]bool {
	possible := map[string]bool{}
	for changed := true; changed; {
		changed = false
		for identity, descriptor := range descriptors {
			if possible[identity] || !hasFactorySeed(descriptor, possible) {
				continue
			}
			possible[identity] = true
			changed = true
		}
	}
	return possible
}
func hasFactorySeed(descriptor factoryDescriptor, possible map[string]bool) bool {
	for _, requirement := range descriptor.requirements {
		if requirement.primitive || possible[requirement.target] {
			return true
		}
	}
	return false
}
func validFactories(descriptors map[string]factoryDescriptor, possible map[string]bool) map[string]bool {
	valid := map[string]bool{}
	for identity, descriptor := range descriptors {
		valid[identity] = descriptor.valid
	}
	for changed := true; changed; {
		changed = false
		for identity, descriptor := range descriptors {
			if !valid[identity] || validFactoryRequirements(descriptor, valid, possible) {
				continue
			}
			valid[identity] = false
			changed = true
		}
	}
	return valid
}
func validFactoryRequirements(descriptor factoryDescriptor, valid, possible map[string]bool) bool {
	for _, requirement := range descriptor.requirements {
		if !validFactoryRequirement(requirement, valid, possible) {
			return false
		}
	}
	return true
}
func validFactoryRequirement(requirement factoryRequirement, valid, possible map[string]bool) bool {
	if !requirement.primitive && !valid[requirement.target] {
		return false
	}
	return requirement.safe || (!requirement.primitive && !possible[requirement.target])
}
func failureFactories(descriptors map[string]factoryDescriptor, valid, possible map[string]bool) map[string]bool {
	failures := map[string]bool{}
	for identity := range descriptors {
		failures[identity] = valid[identity]
	}
	for changed := true; changed; {
		changed = false
		for identity, descriptor := range descriptors {
			if !failures[identity] || failureRequirements(descriptor, failures, possible) {
				continue
			}
			failures[identity] = false
			changed = true
		}
	}
	return failures
}
func failureRequirements(descriptor factoryDescriptor, failures, possible map[string]bool) bool {
	for _, requirement := range descriptor.requirements {
		if !failureRequirement(requirement, failures, possible) {
			return false
		}
	}
	return true
}
func failureRequirement(requirement factoryRequirement, failures, possible map[string]bool) bool {
	if requirement.errorNil {
		return true
	}
	if !requirement.primitive && !possible[requirement.target] {
		return true
	}
	if !requirement.paired {
		return false
	}
	return requirement.primitive || failures[requirement.target]
}

// Isolation is separate from open/fresh state: a closed OS-created handle can
// remain unaliased, but it must still fail the later freshness requirements.
func isolatedFactories(descriptors map[string]factoryDescriptor, possible map[string]bool) map[string]bool {
	isolated := map[string]bool{}
	for identity, descriptor := range descriptors {
		isolated[identity] = descriptor.valid
	}
	for changed := true; changed; {
		changed = false
		for identity, descriptor := range descriptors {
			if !isolated[identity] || isolatedRequirements(descriptor, isolated, possible) {
				continue
			}
			isolated[identity] = false
			changed = true
		}
	}
	return isolated
}
func isolatedRequirements(descriptor factoryDescriptor, isolated, possible map[string]bool) bool {
	for _, requirement := range descriptor.requirements {
		if requirement.primitive {
			if !requirement.isolated {
				return false
			}
			continue
		}
		if !isolated[requirement.target] && possible[requirement.target] {
			return false
		}
	}
	return true
}
