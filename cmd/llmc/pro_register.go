//go:build pro

package main

// This file is included only when building with `-tags pro`.
// It blank-imports the private pro register package so that
// the Pro implementation can register itself via init().

import _ "github.com/LiboWorks/llm-compiler-pro/register"
