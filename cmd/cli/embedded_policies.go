package main

// embeddedPolicyFiles holds policy file contents embedded at build time by the builder.
// When the builder is used to embed policies, it generates _embedded_policies_gen.go
// with an init() function that populates this map with filename->content pairs.
// When empty (the default), no embedded policies are available and the user must
// provide a policy path at runtime.
var embeddedPolicyFiles = map[string]string{}
