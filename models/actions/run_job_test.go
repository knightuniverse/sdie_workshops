// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAggregateJobStatus(t *testing.T) {
	tests := []struct {
		name     string
		jobs     []*ActionRunJob
		expected Status
	}{
		// Empty list
		{
			name:     "empty jobs",
			jobs:     []*ActionRunJob{},
			expected: StatusRunning,
		},
		// Terminal: failure has highest priority
		{
			name: "failure + cancelled",
			jobs: []*ActionRunJob{
				{Status: StatusFailure},
				{Status: StatusCancelled},
			},
			expected: StatusFailure,
		},
		{
			name: "failure + success",
			jobs: []*ActionRunJob{
				{Status: StatusFailure},
				{Status: StatusSuccess},
			},
			expected: StatusFailure,
		},
		// Terminal: cancelled is next priority
		{
			name: "success + cancelled",
			jobs: []*ActionRunJob{
				{Status: StatusSuccess},
				{Status: StatusCancelled},
			},
			expected: StatusCancelled,
		},
		// Terminal: all skipped
		{
			name: "all skipped",
			jobs: []*ActionRunJob{
				{Status: StatusSkipped},
				{Status: StatusSkipped},
			},
			expected: StatusSkipped,
		},
		// Terminal: success + skipped mixed (all terminal, no failure/cancelled)
		{
			name: "success + skipped",
			jobs: []*ActionRunJob{
				{Status: StatusSuccess},
				{Status: StatusSkipped},
			},
			expected: StatusSuccess,
		},
		// Terminal: all success
		{
			name: "all success",
			jobs: []*ActionRunJob{
				{Status: StatusSuccess},
				{Status: StatusSuccess},
			},
			expected: StatusSuccess,
		},
		// Active: all waiting
		{
			name: "all waiting",
			jobs: []*ActionRunJob{
				{Status: StatusWaiting},
				{Status: StatusWaiting},
			},
			expected: StatusWaiting,
		},
		// Active: all blocked
		{
			name: "all blocked",
			jobs: []*ActionRunJob{
				{Status: StatusBlocked},
				{Status: StatusBlocked},
			},
			expected: StatusBlocked,
		},
		// Active: blocked + running (has running, should not be blocked)
		{
			name: "blocked + running",
			jobs: []*ActionRunJob{
				{Status: StatusBlocked},
				{Status: StatusRunning},
			},
			expected: StatusRunning,
		},
		// Active: waiting + blocked (has waiting, should not be blocked)
		{
			name: "waiting + blocked",
			jobs: []*ActionRunJob{
				{Status: StatusWaiting},
				{Status: StatusBlocked},
			},
			expected: StatusRunning,
		},
		// Active: running only
		{
			name: "running only",
			jobs: []*ActionRunJob{
				{Status: StatusRunning},
			},
			expected: StatusRunning,
		},
		// Single job
		{
			name:     "single success",
			jobs:     []*ActionRunJob{{Status: StatusSuccess}},
			expected: StatusSuccess,
		},
		{
			name:     "single waiting",
			jobs:     []*ActionRunJob{{Status: StatusWaiting}},
			expected: StatusWaiting,
		},
		// Active: done jobs do not break allWaiting
		{
			name: "success + waiting",
			jobs: []*ActionRunJob{
				{Status: StatusSuccess},
				{Status: StatusWaiting},
			},
			expected: StatusWaiting,
		},
		{
			name: "skipped + waiting",
			jobs: []*ActionRunJob{
				{Status: StatusSkipped},
				{Status: StatusWaiting},
			},
			expected: StatusWaiting,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aggregateJobStatus(tt.jobs)
			assert.Equal(t, tt.expected, result)
		})
	}
}
