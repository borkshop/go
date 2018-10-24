// Package caskio reads and writes 1KB block B-trees with a content address store.
//
// CASK divides large content into 1 kilobyte blocks that can contain up to 31
// SHA-256 hashes linking to other blocks.
// Each block "retains" these child block links, obliging the content address store
// to retain every block's transitive children.
//
// To model large objects, CASK uses B-trees.
// Each block can have a height property indicating that it has children.
// A height of zero indicates a leaf content block.
// Leaves can have links, but they are not B-tree links.
// A height of one indicates that the block's children are leaves, and so on.
package caskio
