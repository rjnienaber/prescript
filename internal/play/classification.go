package play

import "strings"

// Shape says whether a divergence is about what the program computed or only
// about how it laid the answer out.
//
// The distinction decides where a finding goes rather than how bad it is.
// BASIC's PRINT puts a space before a non-negative number and another after
// it, and `,` tabs to fourteen-column zones; ports get this wrong constantly
// and identically. Filing each one separately would mean several hundred
// near-identical issues against the same repository, which would be ignored,
// reasonably. Formatting divergences belong together, one report per language;
// a semantic one is about a particular program and belongs on its own.
type Shape string

const (
	// Formatting: the two lines are the same once every run of whitespace is
	// collapsed. Nothing the program computed is in dispute.
	Formatting Shape = "formatting"

	// Semantic: they still differ. Different words, different numbers,
	// different case -- something other than layout.
	Semantic Shape = "semantic"

	// UnknownShape: there is nothing to normalise. The step matched by
	// pattern, so the expected side is a regular expression rather than a
	// line, or the run got through every step and went wrong afterwards.
	UnknownShape Shape = ""
)

// DrawUse says what the tape can be made to tell us about whether randomness
// is involved in a divergence at all.
//
// A port that had drawn nothing when it diverged cannot be diverging because
// of the numbers it drew, whatever else is wrong with it. One that drew
// exactly what the reference drew asked for randomness in the same places, so
// the difference is in what it did with the values. One that drew a different
// amount has restructured how it consumes randomness, and no report can say
// whether that is a bug or a legitimate way to write the same program.
type DrawUse string

const (
	// NoDraws: every port in the finding had taken nothing from the tape.
	NoDraws DrawUse = "no-draws"

	// SameDraws: the ports and the reference had drawn the same amount.
	SameDraws DrawUse = "same-draws"

	// DifferentDraws: known counts that disagree -- with the reference, or
	// with each other.
	DifferentDraws DrawUse = "different-draws"

	// UnknownDraws: not enough of the counts are known to say. No tape was
	// played, or the runtime has no shim to replay one.
	UnknownDraws DrawUse = ""
)

// Classification is what can be worked out about a divergence without a person
// reading it. Both axes are computed, neither is a verdict: they say which
// findings can be filed with confidence and which need somebody to look.
type Classification struct {
	Shape Shape
	Draws DrawUse
}

// Tokens is the classification as it appears in a report, in a fixed
// vocabulary and a fixed order, so a corpus of a few hundred runs can be
// counted and sorted by it. What is not known is left out rather than named as
// unknown, because on most runs it is not known and a report that says so on
// every line has stopped carrying information.
func (classification Classification) Tokens() []string {
	var tokens []string
	if classification.Shape != UnknownShape {
		tokens = append(tokens, string(classification.Shape))
	}
	if classification.Draws != UnknownDraws {
		tokens = append(tokens, string(classification.Draws))
	}
	return tokens
}

// classify works out both axes for one finding. reference is what the
// reference run drew, and ports what each of the ports in this finding drew;
// either may be nil, meaning it could not say.
func (group DivergenceGroup) classify(reference *int, ports []*int) Classification {
	return Classification{Shape: group.shape(), Draws: drawUse(reference, ports)}
}

// shape normalises both sides and compares them again.
//
// The received lines are joined before normalising, so a port that broke one
// logical line across two prints is still formatting rather than semantic --
// where the line breaks is layout by any reading.
func (group DivergenceGroup) shape() Shape {
	if group.Matched || group.Redacted {
		return UnknownShape
	}

	if normalise(group.Expected) == normalise(strings.Join(group.Received, " ")) {
		return Formatting
	}
	return Semantic
}

// normalise collapses every run of whitespace to a single space and trims the
// ends, which is the whole of what "formatting" means here. Case is not
// touched: a port that shouts where the reference whispers has changed the
// text, not its layout.
func normalise(line string) string {
	return strings.Join(strings.Fields(line), " ")
}

func drawUse(reference *int, ports []*int) DrawUse {
	drawn, known := agreed(ports)
	if !known {
		// Ports in one finding printed the same thing at the same step, so
		// counts that disagree mean they got there by drawing differently --
		// which is the thing this axis exists to notice, whether or not the
		// reference can say what it drew.
		if disagree(ports) {
			return DifferentDraws
		}
		return UnknownDraws
	}

	if drawn == 0 {
		// True without the reference: a port that has drawn nothing is not
		// diverging over the numbers it drew.
		return NoDraws
	}

	if reference == nil {
		return UnknownDraws
	}

	if *reference == drawn {
		return SameDraws
	}
	return DifferentDraws
}

// agreed is the count every port in the finding reported, and whether they all
// reported one and all reported the same.
func agreed(ports []*int) (int, bool) {
	if len(ports) == 0 || ports[0] == nil {
		return 0, false
	}

	for _, drawn := range ports[1:] {
		if drawn == nil || *drawn != *ports[0] {
			return 0, false
		}
	}
	return *ports[0], true
}

func disagree(ports []*int) bool {
	var seen *int
	for _, drawn := range ports {
		if drawn == nil {
			continue
		}
		if seen != nil && *seen != *drawn {
			return true
		}
		seen = drawn
	}
	return false
}
