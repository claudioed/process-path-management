// Package architecture holds fitness tests (the Go analogue of ArchUnit,
// via github.com/arch-go/arch-go) that enforce the hexagonal/ports-and-adapters
// dependency rule described in this project's README: dependencies point
// inward only, and inbound/outbound adapters never depend on each other.
package architecture

import (
	"testing"

	archgo "github.com/arch-go/arch-go/api"
	"github.com/arch-go/arch-go/api/configuration"
)

const modulePath = "github.com/claudioed/process-path-management"

func TestHexagonalArchitecture(t *testing.T) {
	moduleInfo := configuration.Load(modulePath)

	// arch-go's package-pattern DSL uses '.' as the path-segment separator
	// (mirroring Java package notation), not '/': "**.internal.domain.**"
	// matches any Go import path containing an internal/domain segment,
	// e.g. github.com/claudioed/process-path-management/internal/domain/processpath.

	t.Run("domain depends on nothing internal except domain", func(t *testing.T) {
		rule := &configuration.DependenciesRule{
			Package: "**.internal.domain.**",
			ShouldOnlyDependsOn: &configuration.Dependencies{
				Internal: []string{"**.internal.domain.**"},
			},
		}

		result := archgo.CheckArchitecture(moduleInfo, configuration.Config{
			DependenciesRules: []*configuration.DependenciesRule{rule},
		})

		assertPass(t, result)
	})

	t.Run("application depends only on domain", func(t *testing.T) {
		rule := &configuration.DependenciesRule{
			Package: "**.internal.application.**",
			ShouldOnlyDependsOn: &configuration.Dependencies{
				Internal: []string{
					"**.internal.domain.**",
					"**.internal.application.**",
				},
			},
		}

		result := archgo.CheckArchitecture(moduleInfo, configuration.Config{
			DependenciesRules: []*configuration.DependenciesRule{rule},
		})

		assertPass(t, result)
	})

	t.Run("inbound adapters do not depend on outbound adapters", func(t *testing.T) {
		rule := &configuration.DependenciesRule{
			Package: "**.internal.adapters.inbound.**",
			ShouldNotDependsOn: &configuration.Dependencies{
				Internal: []string{"**.internal.adapters.outbound.**"},
			},
		}

		result := archgo.CheckArchitecture(moduleInfo, configuration.Config{
			DependenciesRules: []*configuration.DependenciesRule{rule},
		})

		assertPass(t, result)
	})

	t.Run("outbound adapters do not depend on inbound adapters", func(t *testing.T) {
		rule := &configuration.DependenciesRule{
			Package: "**.internal.adapters.outbound.**",
			ShouldNotDependsOn: &configuration.Dependencies{
				Internal: []string{"**.internal.adapters.inbound.**"},
			},
		}

		result := archgo.CheckArchitecture(moduleInfo, configuration.Config{
			DependenciesRules: []*configuration.DependenciesRule{rule},
		})

		assertPass(t, result)
	})

	t.Run("only cmd is the composition root wiring every layer", func(t *testing.T) {
		// Nothing under internal/** may import cmd/**: if it did, cmd would
		// no longer be a leaf composition root but a dependency of the very
		// layers it is supposed to wire together.
		rule := &configuration.DependenciesRule{
			Package: "**.internal.**",
			ShouldNotDependsOn: &configuration.Dependencies{
				Internal: []string{"**.cmd.**"},
			},
		}

		result := archgo.CheckArchitecture(moduleInfo, configuration.Config{
			DependenciesRules: []*configuration.DependenciesRule{rule},
		})

		assertPass(t, result)
	})

	// DEVIATION FROM labor-performance (documented, not silently dropped):
	// this service has no analytics data-mesh side (no
	// internal/analytics/report region), so the two ADR-0007-style
	// analytics-isolation rules labor-performance's own arch-fitness suite
	// asserts are omitted here — there is nothing to constrain. If a
	// future analytics side-projection is added to this service, those
	// rules should be added back at that time (see labor-performance's
	// architecture_test.go for the exact shape to mirror).
}

func assertPass(t *testing.T, result *archgo.Result) {
	t.Helper()

	if result.Pass {
		return
	}

	if result.DependenciesRuleResult != nil {
		for _, r := range result.DependenciesRuleResult.Results {
			if !r.Passes {
				t.Errorf("dependency rule %q failed: %+v", r.Description, r.Verifications)
			}
		}
	}

	if result.ContentsRuleResult != nil {
		for _, r := range result.ContentsRuleResult.Results {
			if !r.Passes {
				t.Errorf("contents rule %q failed: %+v", r.Description, r.Verifications)
			}
		}
	}

	t.FailNow()
}
