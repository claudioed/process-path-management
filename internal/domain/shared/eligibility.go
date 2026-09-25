package shared

// Eligibility is a value object declaring the rules a unit of work must
// satisfy to be routed to a given ProcessPath (ADR 0010, the fulfillment
// capability contract). Every field is optional and the zero value
// (Eligibility{}) is fully valid and permissive — "no restrictions
// declared" — which is exactly what every path gets on migration, so
// today's routing behaviour is preserved until an operator narrows it.
//
//   - MaxUnitsPerLine: nil means unbounded; 1 is how a singles path is
//     declared.
//   - RequiredProductAttributes / ExcludedProductAttributes: sets over the
//     product classification vocabulary the fleet already carries on the
//     wire (e.g. "hazmat", "fragile", "giftWrap") — this service does not
//     invent or validate against that vocabulary, exactly like Capability
//     is a plain string this service does not own.
//   - NonSortable: whether this path only accepts non-sortable units.
type Eligibility struct {
	maxUnitsPerLine           *int
	requiredProductAttributes []string
	excludedProductAttributes []string
	nonSortable               bool
}

// NewEligibility constructs an Eligibility value object. There is no
// invariant to enforce beyond copying inputs defensively — every possible
// combination of these fields, including all-zero, is a valid
// declaration.
func NewEligibility(maxUnitsPerLine *int, requiredProductAttributes, excludedProductAttributes []string, nonSortable bool) Eligibility {
	return Eligibility{
		maxUnitsPerLine:           copyIntPtr(maxUnitsPerLine),
		requiredProductAttributes: append([]string(nil), requiredProductAttributes...),
		excludedProductAttributes: append([]string(nil), excludedProductAttributes...),
		nonSortable:               nonSortable,
	}
}

// MaxUnitsPerLine returns a defensive copy of the configured cap, or nil
// for unbounded.
func (e Eligibility) MaxUnitsPerLine() *int { return copyIntPtr(e.maxUnitsPerLine) }

// RequiredProductAttributes returns a defensive copy of the required
// attribute set.
func (e Eligibility) RequiredProductAttributes() []string {
	return append([]string(nil), e.requiredProductAttributes...)
}

// ExcludedProductAttributes returns a defensive copy of the excluded
// attribute set.
func (e Eligibility) ExcludedProductAttributes() []string {
	return append([]string(nil), e.excludedProductAttributes...)
}

// NonSortable reports whether this path only accepts non-sortable units.
func (e Eligibility) NonSortable() bool { return e.nonSortable }

// Equal reports whether e and other declare the same rules — used by
// ProcessPath.Revise's own no-op-change detection, the same convention
// matchPrefix/requiredCapabilities already follow.
func (e Eligibility) Equal(other Eligibility) bool {
	if (e.maxUnitsPerLine == nil) != (other.maxUnitsPerLine == nil) {
		return false
	}
	if e.maxUnitsPerLine != nil && *e.maxUnitsPerLine != *other.maxUnitsPerLine {
		return false
	}
	if e.nonSortable != other.nonSortable {
		return false
	}
	return stringSlicesEqual(e.requiredProductAttributes, other.requiredProductAttributes) &&
		stringSlicesEqual(e.excludedProductAttributes, other.excludedProductAttributes)
}

func copyIntPtr(v *int) *int {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
