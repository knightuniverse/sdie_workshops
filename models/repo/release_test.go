// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/optional"

	"github.com/stretchr/testify/assert"
)

func TestMigrate_InsertReleases(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	a := &Attachment{
		UUID: "a0eebc91-9c0c-4ef7-bb6e-6bb9bd380a12",
	}
	r := &Release{
		Attachments: []*Attachment{a},
	}

	err := InsertReleases(db.DefaultContext, r)
	assert.NoError(t, err)
}

func TestFindReleasesOptions_Keyword(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Base options for repo 1, including tags with real commit IDs
	baseOpts := FindReleasesOptions{
		ListOptions:   db.ListOptions{ListAll: true},
		RepoID:        1,
		IncludeDrafts: true,
		IncludeTags:   true,
		HasSha1:       optional.Some(true),
	}

	// No keyword returns all matching releases
	releases, err := db.Find[Release](db.DefaultContext, baseOpts)
	assert.NoError(t, err)
	assert.NotEmpty(t, releases)
	allCount := len(releases)

	// Keyword matching a specific tag
	opts := baseOpts
	opts.Keyword = "v1.1"
	releases, err = db.Find[Release](db.DefaultContext, opts)
	assert.NoError(t, err)
	for _, r := range releases {
		assert.Contains(t, r.TagName, "v1.1")
	}

	// Keyword matching nothing
	opts = baseOpts
	opts.Keyword = "nonexistent-tag-xyz"
	releases, err = db.Find[Release](db.DefaultContext, opts)
	assert.NoError(t, err)
	assert.Empty(t, releases)

	// Empty keyword returns all (same as no keyword)
	opts = baseOpts
	opts.Keyword = ""
	releases, err = db.Find[Release](db.DefaultContext, opts)
	assert.NoError(t, err)
	assert.Len(t, releases, allCount)

	// Count with keyword matches find results
	opts = baseOpts
	opts.Keyword = "v1.0"
	count, err := db.Count[Release](db.DefaultContext, opts)
	assert.NoError(t, err)
	releases, err = db.Find[Release](db.DefaultContext, opts)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(releases)), count)
}
