// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package cozeloop

import (
	"github.com/alva-ai/cozeloop-go/internal"
)

// Version returns the version of the loop package.
func Version() string {
	return internal.Version()
}
