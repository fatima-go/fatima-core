/*
 * Copyright 2026 github.com/fatima-go
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * @project fatima-core
 * @author dave
 * @date 26. 6. 26.
 */

package infra

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-core/builder"
)

// --- test doubles -----------------------------------------------------------

type recorder struct {
	mu    sync.Mutex
	order []string
}

func (r *recorder) add(s string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.order = append(r.order, s)
}

func (r *recorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// baseComp implements fatima.FatimaComponent only.
type baseComp struct {
	name         string
	rec          *recorder
	shutdownWait time.Duration
}

func (c *baseComp) Initialize() bool { return true }
func (c *baseComp) Bootup()          {}
func (c *baseComp) Shutdown() {
	if c.shutdownWait > 0 {
		time.Sleep(c.shutdownWait)
	}
	if c.rec != nil {
		c.rec.add("shutdown:" + c.name)
	}
}

// lcComp additionally declares a lifecycle timeout.
type lcComp struct {
	*baseComp
	lifecycle time.Duration
}

func (c *lcComp) LifecycleTimeout() time.Duration { return c.lifecycle }

// goawayComp additionally implements fatima.FatimaRuntimeGoaway.
type goawayComp struct {
	*baseComp
	goawayWait time.Duration
}

func (c *goawayComp) Goaway() {
	if c.goawayWait > 0 {
		time.Sleep(c.goawayWait)
	}
	if c.rec != nil {
		c.rec.add("goaway:" + c.name)
	}
}

// goawayLcComp is a goaway target that also declares a lifecycle timeout.
type goawayLcComp struct {
	*goawayComp
	lifecycle time.Duration
}

func (c *goawayLcComp) LifecycleTimeout() time.Duration { return c.lifecycle }

// testConfig is a minimal fatima.Config backed by a map.
type testConfig struct {
	m map[string]string
}

func newTestConfig() *testConfig { return &testConfig{m: make(map[string]string)} }

func (c *testConfig) set(k, v string) *testConfig { c.m[k] = v; return c }

func (c *testConfig) GetValue(key string) (string, bool) {
	v, ok := c.m[key]
	return v, ok
}
func (c *testConfig) GetString(key string) (string, error) {
	if v, ok := c.m[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("not found : %s", key)
}
func (c *testConfig) GetInt(key string) (int, error) {
	v, ok := c.m[key]
	if !ok {
		return 0, fmt.Errorf("not found : %s", key)
	}
	var i int
	_, err := fmt.Sscanf(v, "%d", &i)
	if err != nil {
		return 0, fmt.Errorf("not numeric : %s", key)
	}
	return i, nil
}
func (c *testConfig) GetBool(key string) (bool, error)     { return false, nil }
func (c *testConfig) GetList(key string) ([]string, error) { return nil, nil }

// resetGroups clears the package-level component group slices so each test runs
// against a clean registry.
func resetGroups() {
	compPreInit = make([]fatima.FatimaComponent, 0)
	compGeneral = make([]fatima.FatimaComponent, 0)
	compReader = make([]fatima.FatimaComponent, 0)
	compWriter = make([]fatima.FatimaComponent, 0)
}

// --- resolveGroupTimeout (R2 / AC1, AC2) ------------------------------------

func TestResolveGroupTimeout(t *testing.T) {
	const key = builder.GofatimaComponentLifecycleTimeoutSeconds

	cases := []struct {
		name   string
		config fatima.Config
		want   time.Duration
	}{
		{"nil config falls back to default", nil, 30 * time.Second},
		{"missing key falls back to default", newTestConfig(), 30 * time.Second},
		{"explicit 12 seconds", newTestConfig().set(key, "12"), 12 * time.Second},
		{"zero falls back to default", newTestConfig().set(key, "0"), 30 * time.Second},
		{"negative falls back to default", newTestConfig().set(key, "-5"), 30 * time.Second},
		{"non-numeric falls back to default", newTestConfig().set(key, "abc"), 30 * time.Second},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveGroupTimeout(tc.config)
			if got != tc.want {
				t.Fatalf("resolveGroupTimeout = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- effectiveGroupTimeout (R3 / R4 / AC3) ----------------------------------

func TestEffectiveGroupTimeout(t *testing.T) {
	rec := &recorder{}
	plain := func(n string) fatima.FatimaComponent { return &baseComp{name: n, rec: rec} }
	declared := func(n string, d time.Duration) fatima.FatimaComponent {
		return &lcComp{baseComp: &baseComp{name: n, rec: rec}, lifecycle: d}
	}

	cases := []struct {
		name string
		list []fatima.FatimaComponent
		base time.Duration
		want time.Duration
	}{
		{"empty group uses base", nil, 30 * time.Second, 30 * time.Second},
		{"all undeclared uses base", []fatima.FatimaComponent{plain("a"), plain("b")}, 30 * time.Second, 30 * time.Second},
		{
			// AC3: declared 11s with base 30s -> max = 30s
			"declared below base keeps base",
			[]fatima.FatimaComponent{declared("a", 11*time.Second), plain("b")},
			30 * time.Second,
			30 * time.Second,
		},
		{
			// AC3 reverse: base lowered to 8s, one declares 11s -> 11s
			"declared above base wins",
			[]fatima.FatimaComponent{declared("a", 11*time.Second), plain("b")},
			8 * time.Second,
			11 * time.Second,
		},
		{
			"non-positive declaration falls back to base",
			[]fatima.FatimaComponent{declared("a", 0), declared("b", -3*time.Second)},
			8 * time.Second,
			8 * time.Second,
		},
		{
			// R2/R4: base is the default for undeclared members, NOT a global floor.
			// A group whose every member declares below base waits only for the
			// longest declaration (here a sole member declaring 11s < base 30s).
			"all-declared below base ignores base floor",
			[]fatima.FatimaComponent{declared("a", 11*time.Second)},
			30 * time.Second,
			11 * time.Second,
		},
		{
			// But an undeclared sibling pulls the group back up to base.
			"undeclared sibling restores base",
			[]fatima.FatimaComponent{declared("a", 11*time.Second), plain("b")},
			30 * time.Second,
			30 * time.Second,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := effectiveGroupTimeout(tc.list, tc.base)
			if got != tc.want {
				t.Fatalf("effectiveGroupTimeout = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- shutdownGroup timing (AC4, AC5) ----------------------------------------

// AC4: components that finish quickly return immediately, regardless of timeout.
func TestShutdownGroupReturnsEarlyWhenDone(t *testing.T) {
	resetGroups()
	rec := &recorder{}
	list := []fatima.FatimaComponent{
		&baseComp{name: "fast1", rec: rec},
		&baseComp{name: "fast2", rec: rec},
	}

	start := time.Now()
	shutdownGroup("reader", list, 2*time.Second)
	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected early return, took %v", elapsed)
	}
	if got := len(rec.snapshot()); got != 2 {
		t.Fatalf("expected 2 shutdowns, got %d", got)
	}
}

// AC5: when a component hangs past the timeout, the group stops waiting at ~timeout
// (and the caller proceeds to the next group).
func TestShutdownGroupStopsAtTimeout(t *testing.T) {
	resetGroups()
	rec := &recorder{}
	list := []fatima.FatimaComponent{
		&baseComp{name: "slow", rec: rec, shutdownWait: 800 * time.Millisecond},
	}

	base := 150 * time.Millisecond
	start := time.Now()
	shutdownGroup("reader", list, base)
	elapsed := time.Since(start)

	if elapsed < base {
		t.Fatalf("returned before timeout: %v < %v", elapsed, base)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("waited too long past timeout: %v", elapsed)
	}
}

// Per-component override extends the group wait so a slow-but-declared component
// completes instead of being cut off (effective timeout = max).
func TestShutdownGroupHonorsComponentOverride(t *testing.T) {
	resetGroups()
	rec := &recorder{}
	// base is small (50ms) but the component declares 300ms and needs ~120ms.
	list := []fatima.FatimaComponent{
		&lcComp{
			baseComp:  &baseComp{name: "drainer", rec: rec, shutdownWait: 120 * time.Millisecond},
			lifecycle: 300 * time.Millisecond,
		},
	}

	start := time.Now()
	shutdownGroup("reader", list, 50*time.Millisecond)
	elapsed := time.Since(start)

	if got := rec.snapshot(); len(got) != 1 {
		t.Fatalf("expected drainer to complete, got %v after %v", got, elapsed)
	}
	if elapsed > 300*time.Millisecond {
		t.Fatalf("completed but waited too long: %v", elapsed)
	}
}

// --- shutdownComponent ordering (group sequence) ----------------------------

func TestShutdownComponentGroupOrder(t *testing.T) {
	resetGroups()
	rec := &recorder{}
	compReader = []fatima.FatimaComponent{&baseComp{name: "reader", rec: rec}}
	compGeneral = []fatima.FatimaComponent{&baseComp{name: "general", rec: rec}}
	compWriter = []fatima.FatimaComponent{&baseComp{name: "writer", rec: rec}}
	compPreInit = []fatima.FatimaComponent{&baseComp{name: "preinit", rec: rec}}

	shutdownComponent("test-proc", 1*time.Second)

	want := []string{"shutdown:reader", "shutdown:general", "shutdown:writer", "shutdown:preinit"}
	got := rec.snapshot()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order mismatch at %d: got %v, want %v", i, got, want)
		}
	}
}

// --- goaway (AC6) -----------------------------------------------------------

// Only FatimaRuntimeGoaway implementers are signaled; the same effective-timeout
// rule applies (AC6: consistent goaway/shutdown behavior).
func TestGoawayGroupTargetsOnly(t *testing.T) {
	resetGroups()
	rec := &recorder{}
	list := []fatima.FatimaComponent{
		&baseComp{name: "plain", rec: rec}, // not a goaway target
		&goawayComp{baseComp: &baseComp{name: "target", rec: rec}},
	}

	count := goawayGroup("reader", list, 1*time.Second)

	if count != 1 {
		t.Fatalf("expected 1 goaway target, got %d", count)
	}
	got := rec.snapshot()
	if len(got) != 1 || got[0] != "goaway:target" {
		t.Fatalf("expected only target signaled, got %v", got)
	}
}

// R4: effective timeout is computed over the goaway target set only. A non-target
// component's declared timeout must not extend the group wait.
func TestGoawayGroupIgnoresNonTargetTimeout(t *testing.T) {
	resetGroups()
	rec := &recorder{}
	list := []fatima.FatimaComponent{
		// Non-target declares a huge timeout but is NOT signaled -> must be ignored.
		&lcComp{baseComp: &baseComp{name: "nontarget", rec: rec}, lifecycle: 10 * time.Second},
		// Target hangs; only base applies since it declares nothing.
		&goawayComp{baseComp: &baseComp{name: "target", rec: rec}, goawayWait: 800 * time.Millisecond},
	}

	base := 150 * time.Millisecond
	start := time.Now()
	goawayGroup("reader", list, base)
	elapsed := time.Since(start)

	if elapsed < base {
		t.Fatalf("returned before timeout: %v", elapsed)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("non-target timeout leaked into group wait: %v", elapsed)
	}
}

// A goaway target that declares its own timeout extends the group wait so it can
// finish its drain.
func TestGoawayGroupHonorsTargetOverride(t *testing.T) {
	resetGroups()
	rec := &recorder{}
	list := []fatima.FatimaComponent{
		&goawayLcComp{
			goawayComp: &goawayComp{baseComp: &baseComp{name: "drainer", rec: rec}, goawayWait: 120 * time.Millisecond},
			lifecycle:  300 * time.Millisecond,
		},
	}

	start := time.Now()
	goawayGroup("reader", list, 50*time.Millisecond)
	elapsed := time.Since(start)

	got := rec.snapshot()
	if len(got) != 1 || got[0] != "goaway:drainer" {
		t.Fatalf("expected drainer to complete, got %v after %v", got, elapsed)
	}
}

// A target declaring a timeout BELOW base caps the group at its declaration, not at
// base (R2/R4: base is the default for undeclared, not a floor). A hanging target is
// cut off at its own declared bound.
func TestGoawayGroupDeclarationBelowBaseCaps(t *testing.T) {
	resetGroups()
	rec := &recorder{}
	list := []fatima.FatimaComponent{
		&goawayLcComp{
			goawayComp: &goawayComp{baseComp: &baseComp{name: "capped", rec: rec}, goawayWait: 800 * time.Millisecond},
			lifecycle:  150 * time.Millisecond,
		},
	}

	base := 400 * time.Millisecond
	start := time.Now()
	goawayGroup("reader", list, base)
	elapsed := time.Since(start)

	if elapsed < 150*time.Millisecond {
		t.Fatalf("returned before declared bound: %v", elapsed)
	}
	if elapsed >= base {
		t.Fatalf("base leaked as floor; waited %v (>= base %v)", elapsed, base)
	}
}

// goawayComponent skips groups with no targets and reports zero work.
func TestGoawayComponentNoTargets(t *testing.T) {
	resetGroups()
	rec := &recorder{}
	compGeneral = []fatima.FatimaComponent{&baseComp{name: "plain", rec: rec}}

	goawayComponent(1 * time.Second) // must not panic and must not signal anyone

	if got := rec.snapshot(); len(got) != 0 {
		t.Fatalf("expected no goaway calls, got %v", got)
	}
}
