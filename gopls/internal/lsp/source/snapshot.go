// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package source

import (
	"context"
	"fmt"

	"golang.org/x/tools/gopls/internal/file"
	"golang.org/x/tools/gopls/internal/lsp/cache"
	"golang.org/x/tools/gopls/internal/lsp/cache/metadata"
	"golang.org/x/tools/gopls/internal/lsp/cache/parsego"
	"golang.org/x/tools/gopls/internal/lsp/protocol"
)

// NarrowestMetadataForFile returns metadata for the narrowest package
// (the one with the fewest files) that encloses the specified file.
// The result may be a test variant, but never an intermediate test variant.
func NarrowestMetadataForFile(ctx context.Context, snapshot *cache.Snapshot, uri protocol.DocumentURI) (*Metadata, error) {
	metas, err := snapshot.MetadataForFile(ctx, uri)
	if err != nil {
		return nil, err
	}
	metadata.RemoveIntermediateTestVariants(&metas)
	if len(metas) == 0 {
		return nil, fmt.Errorf("no package metadata for file %s", uri)
	}
	return metas[0], nil
}

// NarrowestPackageForFile is a convenience function that selects the narrowest
// non-ITV package to which this file belongs, type-checks it in the requested
// mode (full or workspace), and returns it, along with the parse tree of that
// file.
//
// The "narrowest" package is the one with the fewest number of files that
// includes the given file. This solves the problem of test variants, as the
// test will have more files than the non-test package.
//
// An intermediate test variant (ITV) package has identical source to a regular
// package but resolves imports differently. gopls should never need to
// type-check them.
//
// Type-checking is expensive. Call snapshot.ParseGo if all you need is a parse
// tree, or snapshot.MetadataForFile if you only need metadata.
func NarrowestPackageForFile(ctx context.Context, snapshot *cache.Snapshot, uri protocol.DocumentURI) (*cache.Package, *ParsedGoFile, error) {
	return selectPackageForFile(ctx, snapshot, uri, func(metas []*Metadata) *Metadata { return metas[0] })
}

// WidestPackageForFile is a convenience function that selects the widest
// non-ITV package to which this file belongs, type-checks it in the requested
// mode (full or workspace), and returns it, along with the parse tree of that
// file.
//
// The "widest" package is the one with the most number of files that includes
// the given file. Which is the test variant if one exists.
//
// An intermediate test variant (ITV) package has identical source to a regular
// package but resolves imports differently. gopls should never need to
// type-check them.
//
// Type-checking is expensive. Call snapshot.ParseGo if all you need is a parse
// tree, or snapshot.MetadataForFile if you only need metadata.
func WidestPackageForFile(ctx context.Context, snapshot *cache.Snapshot, uri protocol.DocumentURI) (*cache.Package, *ParsedGoFile, error) {
	return selectPackageForFile(ctx, snapshot, uri, func(metas []*Metadata) *Metadata { return metas[len(metas)-1] })
}

func selectPackageForFile(ctx context.Context, snapshot *cache.Snapshot, uri protocol.DocumentURI, selector func([]*Metadata) *Metadata) (*cache.Package, *ParsedGoFile, error) {
	metas, err := snapshot.MetadataForFile(ctx, uri)
	if err != nil {
		return nil, nil, err
	}
	metadata.RemoveIntermediateTestVariants(&metas)
	if len(metas) == 0 {
		return nil, nil, fmt.Errorf("no package metadata for file %s", uri)
	}
	md := selector(metas)
	pkgs, err := snapshot.TypeCheck(ctx, md.ID)
	if err != nil {
		return nil, nil, err
	}
	pkg := pkgs[0]
	pgf, err := pkg.File(uri)
	if err != nil {
		return nil, nil, err // "can't happen"
	}
	return pkg, pgf, err
}

// A FileSource maps URIs to FileHandles.
type FileSource interface {
	// ReadFile returns the FileHandle for a given URI, either by
	// reading the content of the file or by obtaining it from a cache.
	//
	// Invariant: ReadFile must only return an error in the case of context
	// cancellation. If ctx.Err() is nil, the resulting error must also be nil.
	ReadFile(ctx context.Context, uri protocol.DocumentURI) (file.Handle, error)
}

type ParsedGoFile = parsego.File

const (
	ParseHeader = parsego.ParseHeader
	ParseFull   = parsego.ParseFull
)

type (
	PackageID   = metadata.PackageID
	PackagePath = metadata.PackagePath
	PackageName = metadata.PackageName
	ImportPath  = metadata.ImportPath
	Metadata    = metadata.Metadata
)

type unit = struct{}
