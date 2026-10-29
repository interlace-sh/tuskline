// SPDX-License-Identifier: Apache-2.0

// Package analyser turns collected facts into findings. Each finding has
// the evidence, the check that produced it and a fix. It does not open
// database connections; callers pass the numbers.
package analyser
