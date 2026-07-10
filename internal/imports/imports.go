// Package imports ensures all module packages are imported so their init()
// functions run and register themselves with the module registry.
//
// Each import is guarded by the same build tag as the module itself,
// so modules excluded at build time won't be imported either.
package imports

// Module imports — each is conditionally compiled via build tags.
// The blank imports trigger the init() registration in each module package.
