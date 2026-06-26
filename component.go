/*
 * Copyright 2023 github.com/fatima-go
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
 * @date 23. 4. 12. 오후 1:41
 */

package fatima

import "time"

type FatimaComponentType int

const (
	COMP_PRE_INIT FatimaComponentType = iota
	COMP_GENERAL
	COMP_READER
	COMP_WRITER
)

type FatimaComponent interface {
	Initialize() bool
	Bootup()
	Shutdown()
}

type FatimaIOReader interface {
	StartListening()
}

type FatimaComponentTypeOrder interface {
	GetType() FatimaComponentType
}

// FatimaComponentLifecycleTimeout is an optional interface a FatimaComponent may
// implement to declare how long it needs during Goaway()/Shutdown() before the
// orchestrator stops waiting for it. Components that do not implement it fall back
// to the config default (gofatima.component.lifecycle.timeout.seconds, default 30s).
// A non-positive return value is also treated as "not declared" and falls back to
// the config default.
type FatimaComponentLifecycleTimeout interface {
	LifecycleTimeout() time.Duration
}
